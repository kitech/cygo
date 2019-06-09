#include "noropriv.h"

#include <event2/event.h>
#include <event2/thread.h>

// 由于 hook中没有hook epoll_wait, epoll_create,
// 所以在这是可以使用libev/libuv。
// 如果以后hook，则这的实现无效了。看似也不需要hook epoll

// typedef struct ev_loop ev_loop;

#define EV_IO EV_READ|EV_WRITE|EV_CLOSED
#define EV_TIMER EV_TIMEOUT

typedef struct netpoller {
    struct event_base * loop;
    HashTable* watchers; // ev_watcher* => goroutine*
    mtx_t evmu;
} netpoller;

static netpoller* gnpl__ = 0;

void netpoller_use_threads() {
    evthread_use_pthreads();
    // or evthread_use_windows_threads()
}

netpoller* netpoller_new() {
    assert(gnpl__ == 0);
    netpoller* np = (netpoller*)calloc(1, sizeof(netpoller));
    np->loop = event_base_new();
    assert(np->loop != 0);

    hashtable_new(&np->watchers);

    gnpl__ = np;
    return np;
}

void netpoller_loop() {
    netpoller* np = gnpl__;
    assert(np != 0);

    for (;;) {
        // int rv = event_base_dispatch(np->loop);
        int flags = EVLOOP_NO_EXIT_ON_EMPTY;
        // flags = 0;
        int rv = event_base_loop(np->loop, flags);
        linfo("ohno, rv=%d\n", rv);
    }
    assert(1==2);
}

typedef struct evdata {
    int evtyp;
    void* data;
    int ytype;
    int fd;
    struct timeval tv;
    struct event* evt;
} evdata;

static void atstgc_finalizer_fn(evdata* obj, void* cbdata) {
    linfo("finilize obj %p, %p\n", obj, cbdata);
}
// TODO seems norogc has some problem?
// 难道说可能是libevent也开了自己的线程？无
// switch to manual calloc can fix the problem: because GC_malloc return the same addr
// 发现 evdata 的finalize早于真正需要释放它的时间？而在 netpoller_readfd()中加一个log顺序就变了？
// 难道是goroutine yield之后，被GC认为是没用了？应该怎么测试呢？
evdata* evdata_new(int evtyp, void* data) {
    assert(evtyp >= 0);

    netpoller* np = gnpl__;
    // evdata* d = noro_malloc_st(evdata);
    evdata* d = calloc(1, sizeof(evdata));
    d->evtyp = evtyp;
    d->data = data;
    // GC_register_finalizer(d, atstgc_finalizer_fn, nilptr, nilptr, nilptr);
    return d;
}
void evdata_free(evdata* d) {
    // noro_free(d);
    free(d);
}

extern void noro_processor_resume_some(void* cbdata, int ytype);

// common version callback, support ev_io, ev_timer
static
void netpoller_evwatcher_cb(evutil_socket_t fd, short events, void* arg) {
    evdata* d = (evdata*)arg;
    assert(d != 0);
    int ytype = d->ytype;
    void* dd = d->data;
    struct event* evt = d->evt;

    switch (d->evtyp) {
    case EV_TIMER:
        // evtimer_del(d->evt);
        break;
    case EV_IO:
        // event_del(d->evt);
        break;
    default:
        linfo("wtf fd=%d %d %d\n", fd, d->evtyp, d->ytype);
        assert(1==2);
    }

    // if event_del then event_free, it crash
    // if direct event_free, it ok.
    // because non-persist event already run event_del by loop itself

    goroutine *gr = dd;
    // linfo("before release d=%p\n", d);
    if (d->evtyp == EV_TIMER && fd != -1) {
        linfo("evwoke ev=%d fd=%d(%d) ytype=%d=%s %p grid=%d, mcid=%d d=%p\n",
              events, fd, fd, ytype, yield_type_name(ytype), dd, gr->id, gr->mcid, d);
        assert(fd == -1);
    }
    event_free(evt);
    evdata_free(d);
    noro_processor_resume_some(dd, ytype);
}

static
void netpoller_readfd(int fd, int ytype, void* gr) {
    netpoller* np = gnpl__;
    evdata* d = evdata_new(EV_IO, gr);
    d->ytype = ytype;
    d->fd = fd;
    // mtx_lock(&np->evmu);
    struct event* evt = event_new(np->loop, fd, EV_READ|EV_CLOSED, netpoller_evwatcher_cb, d);
    d->evt = evt;
    event_add(evt, 0);
    // mtx_unlock(&np->evmu);
    if (d != nilptr) {
        // linfo("event_add d=%p\n", d);
    }
}

static
void netpoller_writefd(int fd, int ytype, void* gr) {
    netpoller* np = gnpl__;
    evdata* d = evdata_new(EV_IO, gr);
    d->ytype = ytype;
    d->fd = fd;
    // mtx_lock(&np->evmu);
    struct event* evt = event_new(np->loop, fd, EV_WRITE|EV_CLOSED, netpoller_evwatcher_cb, d);
    d->evt = evt;
    event_add(evt, 0);
    // mtx_unlock(&np->evmu);
}

static
void netpoller_timer(long ns, int ytype, void* gr) {
    netpoller* np = gnpl__;

    evdata* d = evdata_new(EV_TIMER, gr);
    d->ytype = ytype;
    d->fd = ns;
    d->tv.tv_sec = ns/1000000000;
    d->tv.tv_usec = ns/1000 % 1000000;
    // mtx_lock(&np->evmu);
    struct event* tmer = evtimer_new(np->loop, netpoller_evwatcher_cb, d);
    // struct event* tmer = event_new(np->loop, -1, 0, netpoller_evwatcher_cb, d);
    d->evt = tmer;
    evtimer_add(tmer, &d->tv);
    // mtx_unlock(&np->evmu);
    // linfo("timer add d=%p\n", d);
}

// when ytype is SLEEP/USLEEP/NANOSLEEP, fd is the nanoseconds
void netpoller_yieldfd(int fd, int ytype, void* gr) {
    assert(ytype > YIELD_TYPE_NONE);
    assert(ytype < YIELD_TYPE_MAX);

    struct timeval tv = {0, 123};
    switch (ytype) {
    case YIELD_TYPE_SLEEP: case YIELD_TYPE_USLEEP: case YIELD_TYPE_NANOSLEEP:
        // event_base_loopbreak(gnpl__->loop);
        // event_base_loopexit(gnpl__->loop, &tv);
        break;
    }

    long ns = 0;
    switch (ytype) {
    case YIELD_TYPE_SLEEP:
        ns = (long)fd*1000000000;
        netpoller_timer(ns, ytype, gr);
        break;
    case YIELD_TYPE_USLEEP:
        ns = (long)fd*1000;
        netpoller_timer(ns, ytype, gr);
        break;
    case YIELD_TYPE_NANOSLEEP:
        ns = fd;
        netpoller_timer(ns, ytype, gr);
        break;
    case YIELD_TYPE_CHAN_SEND:
        assert(1==2);// cannot process this type
        netpoller_timer(1000, ytype, gr);
        break;
    case YIELD_TYPE_CHAN_RECV:
        assert(1==2);// cannot process this type
        netpoller_timer(1000, ytype, gr);
        break;
    case YIELD_TYPE_CONNECT: case YIELD_TYPE_WRITE: case YIELD_TYPE_WRITEV:
    case YIELD_TYPE_SEND: case YIELD_TYPE_SENDTO: case YIELD_TYPE_SENDMSG:
        netpoller_writefd(fd, ytype, gr);
        break;
    // case YIELD_TYPE_READ: case YIELD_TYPE_READV:
    // case YIELD_TYPE_RECV: case YIELD_TYPE_RECVFROM: case YIELD_TYPE_RECVMSG:
    default:
        // linfo("add reader fd=%d ytype=%d=%s\n", fd, ytype, yield_type_name(ytype));
        assert(fd >= 0);
        netpoller_readfd(fd, ytype, gr);
        break;
    }

}
