
/* #include <event2/event.h> */
/* #include <event2/thread.h> */
/* #include <event2/dns.h> */

#include <sys/types.h>
#include <sys/socket.h>
#include <netdb.h>
#include <sys/timerfd.h>

#include "picoev.h"

#include "coronapriv.h"

// 由于 hook中没有hook epoll_wait, epoll_create,
// 所以在这是可以使用libev/libuv/picoev。
// 如果以后hook，则这的实现无效了。看似也不需要hook epoll

#define EV_IO PICOEV_READ|PICOEV_WRITE
#define EV_TIMER EV_IO*2
#define EV_DNS_RESOLV 0

// TODO thread mutex

typedef struct netpoller {
    picoev_loop *loop;
    int tmerfd;
    HashTable* tmers; // fd => evdata
    // struct evdns_base* dnsbase;
    HashTable* watchers; // ev_watcher* => fiber*
    pmutex_t evmu;
} netpoller;

static netpoller* gnpl__ = 0;

static void netpoller_logcb(int severity, const char *msg) {
    linfo("lvl=%d msg=%s\n", severity, msg);
}
void netpoller_use_threads() {
    // evthread_use_pthreads();
    // or evthread_use_windows_threads()
    // event_set_mem_functions(crn_gc_malloc, crn_gc_realloc, crn_gc_free);
    // event_set_log_callback(netpoller_logcb);
}

netpoller* netpoller_new() {
    assert(gnpl__ == 0);
    picoev_init(1234);
    netpoller* np = (netpoller*)crn_gc_malloc(sizeof(netpoller));
    np->loop = picoev_create_loop(3456);
    // np->loop = event_base_new();
    assert(np->loop != 0);
    np->tmerfd = timerfd_create(CLOCK_REALTIME, TFD_NONBLOCK | TFD_CLOEXEC);
    // np->dnsbase = evdns_base_new(np->loop, 1);

    hashtable_new(&np->tmers);
    // hashtable_new(&np->watchers);

    gnpl__ = np;
    return np;
}

void netpoller_loop() {
    netpoller* np = gnpl__;
    assert(np != 0);

    for (;;) {
        int rv = picoev_loop_once(np->loop, 100);
        if (0) {
            linfo("ohno, rv=%d\n", rv);
        }
    }
    assert(1==2);
}

typedef struct evdata {
    int evtyp;
    void* data; // fiber*
    int grid;
    int mcid;
    int ytype;
    long fd; // fd or ns or hostname
    void** out; //
    int *errcode;
    struct timeval tv;
    struct event* evt;
} evdata;
typedef struct evdata2 {
    int evtyp;
    evdata* dr;
    evdata* dw;
    evdata* dt;
} evdata2;

static void atstgc_finalizer_fn(evdata* obj, void* cbdata) {
    linfo("finilize obj %p, %p\n", obj, cbdata);
}
// TODO seems coronagc has some problem?
// 难道说可能是libevent也开了自己的线程？无
// switch to manual calloc can fix the problem: because GC_malloc return the same addr
// 发现 evdata 的finalize早于真正需要释放它的时间？而在 netpoller_readfd()中加一个log顺序就变了？
// 难道是fiber yield之后，认为没用了被GC？应该怎么测试呢？
evdata* evdata_new(int evtyp, void* data) {
    assert(evtyp >= 0);

    netpoller* np = gnpl__;
    evdata* d = crn_gc_malloc(sizeof(evdata));
    d->evtyp = evtyp;
    d->data = data;
    // GC_register_finalizer(d, atstgc_finalizer_fn, nilptr, nilptr, nilptr);
    return d;
}
void evdata_free(evdata* d) {
    crn_gc_free(d);
}
evdata2* evdata2_new(int evtyp) {
    netpoller* np = gnpl__;
    evdata2* d = crn_gc_malloc(sizeof(evdata2));
    d->evtyp = evtyp;
    // GC_register_finalizer(d, atstgc_finalizer_fn, nilptr, nilptr, nilptr);
    return d;
}

extern void crn_procer_resume_one(void* cbdata, int ytype, int grid, int mcid);

// common version callback, support ev_io, ev_timer
static
void netpoller_picoev_globcb(picoev_loop* loop, int fd, int events, void* cbarg) {
    evdata2* d2 = (evdata2*)cbarg;
    assert(d2 != 0);
    evdata* d = 0;
    void* dd = 0; // = d->data;
    int ytype = 0; // = d->ytype;
    int grid = 0; //= d->grid;
    int mcid = 0; // d->mcid;
    // struct event* evt = d->evt;

    int newev = 0;
    int rv = 0;
    switch (d2->evtyp) {
    case EV_TIMER:
        rv = picoev_del(loop, fd);
        close(fd);
        d = d2->dt;
        d2->dt = 0;
        break;
    case EV_IO:
        if ((events & PICOEV_READ) && (events & PICOEV_WRITE)) {
            assert(1==2);
        }
        if (events & PICOEV_READ) {
            d = d2->dr;
            d2->dr = 0;
        }else if (events & PICOEV_WRITE) {
            d = d2->dw;
            d2->dw = 0;
        }else{
            assert(1==2);
        }
        if (d2->dr != 0) { newev |= PICOEV_READ; }
        if (d2->dw != 0) { newev |= PICOEV_READ; }
        if (newev == 0) {
            rv = picoev_del(loop, fd);
        }else{
            rv = picoev_set_events(loop, fd, newev);
        }
        break;
    default:
        linfo("wtf fd=%d %d %d\n", fd, d->evtyp, d->ytype);
        assert(1==2);
    }
    dd = d->data;
    ytype = d->ytype;
    grid = d->grid;
    mcid = d->mcid;

    fiber *gr = dd;
    // linfo("before release d=%p\n", d);
    if (d->evtyp == EV_TIMER && fd != -1 && 0) { // use for check data mismatch case
        linfo("evwoke ev=%d fd=%d(%d) ytype=%d=%s %p grid=%d, mcid=%d d=%p\n",
              events, fd, fd, ytype, yield_type_name(ytype), dd, gr->id, gr->mcid, d);
        // assert(fd == -1);
    }
    evdata_free(d);
    // evdata2_free(d2);
    crn_procer_resume_one(dd, ytype, grid, mcid);
}


extern void crn_pre_gclock_proc(const char* funcname);
extern void crn_post_gclock_proc(const char* funcname);

static
void netpoller_readfd(int fd, int ytype, fiber* gr) {
    netpoller* np = gnpl__;
    evdata* d = evdata_new(EV_IO, gr);
    d->grid = gr->id;
    d->mcid = gr->mcid;
    d->ytype = ytype;
    d->fd = fd;

    // struct event* evt = event_new(np->loop, fd, EV_READ|EV_CLOSED, netpoller_evwatcher_cb, d);
    // d->evt = evt;
    crn_pre_gclock_proc(__func__);
    int rv = 0;
    if (picoev_is_active(np->loop, fd)) {
        int ev = picoev_get_events(np->loop, fd);
        evdata2* d2 = picoev.fds[fd].cb_arg;
        d2->dr = d;
        ev = ev | PICOEV_READ;
        rv = picoev_set_events(np->loop, fd, ev);
    }else{
        evdata2* d2 = evdata2_new(EV_IO);
        d2->dr = d;
        rv = picoev_add(np->loop, fd, PICOEV_READ,0, netpoller_picoev_globcb, d2);
    }
    crn_post_gclock_proc(__func__);
    if (rv != 0) {
        lwarn("add error %d %d %d\n", rv, fd, gr->id);
        // evdata2_free(d2);
        evdata_free(d);
        // assert(rv == 0);
        return;
    }

    if (d != nilptr) {
        // linfo("event_add d=%p fd=%d ytype=%d rv=%d\n", d, fd, ytype, rv);
    }
}


// why hang forever when send?
// yield fd=13, ytype=10, mcid=5, grid=5
static
void netpoller_writefd(int fd, int ytype, fiber* gr) {
    netpoller* np = gnpl__;
    evdata* d = evdata_new(EV_IO, gr);
    d->grid = gr->id;
    d->mcid = gr->mcid;
    d->ytype = ytype;
    d->fd = fd;

    crn_pre_gclock_proc(__func__);
    int rv = 0;
    if (picoev_is_active(np->loop, fd)) {
        int ev = picoev_get_events(np->loop, fd);
        evdata2* d2 = picoev.fds[fd].cb_arg;
        d2->dr = d;
        ev = ev | PICOEV_WRITE;
        rv = picoev_set_events(np->loop, fd, ev);
    }else{
        evdata2* d2 = evdata2_new(EV_IO);
        d2->dw = d;
        rv = picoev_add(np->loop, fd, PICOEV_WRITE, 0, netpoller_picoev_globcb, d2);
    }
    crn_post_gclock_proc(__func__);
    if (rv != 0) {
        lwarn("add error %d %d %d\n", rv, fd, gr->id);
        // evdata2_free(evt);
        evdata_free(d);
        // assert(rv == 0);
        return;
    }

    // linfo("evwrite add d=%p %ld\n", d, fd);
}

static
void netpoller_timer(long ns, int ytype, fiber* gr) {
    netpoller* np = gnpl__;

    evdata* d = evdata_new(EV_TIMER, gr);
    d->grid = gr->id;
    d->mcid = gr->mcid;
    d->ytype = ytype;
    d->fd = ns;
    d->tv.tv_sec = ns/1000000000;
    d->tv.tv_usec = ns/1000 % 1000000;

    int tmfd = timerfd_create(CLOCK_REALTIME, TFD_NONBLOCK | TFD_CLOEXEC);
    d->fd = tmfd;

    struct timespec ts;
    ts.tv_sec = d->tv.tv_sec;
    ts.tv_nsec = d->tv.tv_usec*1000;
    struct itimerspec its;
    its.it_interval = ts;
    its.it_value = ts;
    int rv0 = timerfd_settime(tmfd, 0, &its, 0);

    evdata2* d2 = evdata2_new(EV_TIMER);
    d2->dt = d;
    crn_pre_gclock_proc(__func__);
    int rv = picoev_add(np->loop, tmfd, PICOEV_READ, 0, netpoller_picoev_globcb, d2);
    crn_post_gclock_proc(__func__);
    if (rv != 0) {
        lwarn("add error %d %ld %d\n", rv, ns, gr->id);
        // evdata2_free(tmer);
        evdata_free(d);
        // assert(rv == 0);
        return;
    }

    // linfo("timer add d=%p %ld\n", d, ns);
}

// what to do
static struct addrinfo* netpoller_dump_addrinfo(struct evutil_addrinfo* addr) {
    assert(1==2);
    return 0;
}
extern bool crn_procer_resume_prechk(void* gr_, int ytype, int grid, int mcid);
// what to do
/*
static
void evdns_resolv_cbproc(int errcode, struct evutil_addrinfo *addr, void *ptr)
{
}
*/

// what to do
void* netpoller_dnsresolv(const char* hostname, int ytype, fiber* gr, struct addrinfo** addr, int *errcode) {
    assert(1==2);
    return 0;
}

// when ytype is SLEEP/USLEEP/NANOSLEEP, fd is the nanoseconds
void netpoller_yieldfd(long fd, int ytype, fiber* gr) {
    assert(ytype > YIELD_TYPE_NONE);
    assert(ytype < YIELD_TYPE_MAX);

    struct timeval tv = {0, 123};
    switch (ytype) {
    case YIELD_TYPE_SLEEP: case YIELD_TYPE_MSLEEP:
    case YIELD_TYPE_USLEEP: case YIELD_TYPE_NANOSLEEP:
        // event_base_loopbreak(gnpl__->loop);
        // event_base_loopexit(gnpl__->loop, &tv);
        break;
    }
    // linfo("fd=%ld, ytype=%d\n", fd, ytype);

    long ns = 0;
    switch (ytype) {
    case YIELD_TYPE_SLEEP:
        ns = fd*1000000000;
        netpoller_timer(ns, ytype, gr);
        break;
    case YIELD_TYPE_MSLEEP:
        ns = fd*1000000;
        netpoller_timer(ns, ytype, gr);
        break;
    case YIELD_TYPE_USLEEP:
        ns = fd*1000;
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
    case YIELD_TYPE_GETADDRINFO:
    //    netpoller_dnsresolv((char*)fd, ytype, gr);
        break;
    default:
        // linfo("add reader fd=%d ytype=%d=%s\n", fd, ytype, yield_type_name(ytype));
        assert(fd >= 0);
        netpoller_readfd(fd, ytype, gr);
        break;
    }

}
