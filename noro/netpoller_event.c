#include "noropriv.h"

#include <event2/event.h>
#include <event2/thread.h>

// 由于 hook中没有hook epoll_wait, epoll_create,
// 所以在这是可以使用libev/libuv。
// 如果以后hook，则这的实现无效了。看似也不需要hook epoll

#define HKDEBUG 1
#define linfo(fmt, ...)                                                 \
    do { if (HKDEBUG) fprintf(stderr, "%s:%d:%s ", __FILE__, __LINE__, __FUNCTION__); } while (0); \
    do { if (HKDEBUG) fprintf(stderr, fmt, __VA_ARGS__); } while (0) ;  \
    do { if (HKDEBUG) fflush(stderr); } while (0) ;

// typedef struct ev_loop ev_loop;

#define EV_IO EV_READ|EV_WRITE
#define EV_TIMER EV_TIMEOUT

typedef struct netpoller {
    struct event_base * loop;
    HashTable* watchers; // ev_watcher* => goroutine*
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
    struct timeval tv;
    struct event* evt;
} evdata;
evdata* evdata_new(int evtyp, void* data) {
    evdata* d = noro_malloc_st(evdata);
    d->evtyp = evtyp;
    d->data = data;
    return d;
}
void evdata_free(evdata* d) { noro_free(d); }

extern void noro_processor_resume_some(void* cbdata);

// common version callback, support ev_io, ev_timer
static
void netpoller_evwatcher_cb(evutil_socket_t fd, short events, void* arg) {
    evdata* d = (evdata*)arg;
    assert(d != 0);

    switch (d->evtyp) {
    case EV_TIMER:
        // evtimer_del(d->evt);
        break;
    case EV_IO:
        // event_del(d->evt);
        break;
    default:
        assert(1==2);
    }

    // if event_del then event_free, it crash
    // if direct event_free, it ok.
    // because non-persist event already run event_del by loop itself

    void* dd = d->data;
    event_free(d->evt);
    evdata_free(d);
    noro_processor_resume_some(dd);
}

static
void netpoller_readfd(int fd, void* gr) {
    netpoller* np = gnpl__;
    evdata* d = evdata_new(EV_IO, gr);
    struct event* evt = event_new(np->loop, fd, EV_READ, netpoller_evwatcher_cb, d);
    d->evt = evt;
    event_add(evt, 0);
}

static
void netpoller_writefd(int fd, void* gr) {
    netpoller* np = gnpl__;
    evdata* d = evdata_new(EV_IO, gr);
    struct event* evt = event_new(np->loop, fd, EV_WRITE, netpoller_evwatcher_cb, d);
    d->evt = evt;
    event_add(evt, 0);
}

static
void netpoller_timer(long ns, void* gr) {
    netpoller* np = gnpl__;

    evdata* d = evdata_new(EV_TIMER, gr);
    d->tv.tv_sec = ns/1000000000;
    d->tv.tv_usec = ns/1000 % 1000000;
    struct event* tmer = evtimer_new(np->loop, netpoller_evwatcher_cb, d);
    d->evt = tmer;
    evtimer_add(tmer, &d->tv);
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
        netpoller_timer(ns, gr);
        break;
    case YIELD_TYPE_USLEEP:
        ns = (long)fd*1000;
        netpoller_timer(ns, gr);
        break;
    case YIELD_TYPE_NANOSLEEP:
        ns = fd;
        netpoller_timer(ns, gr);
        break;
    case YIELD_TYPE_CHAN_SEND:
        assert(1==2);// cannot process this type
        netpoller_timer(1000, gr);
        break;
    case YIELD_TYPE_CHAN_RECV:
        assert(1==2);// cannot process this type
        netpoller_timer(1000, gr);
        break;
    case YIELD_TYPE_CONNECT: case YIELD_TYPE_WRITE: case YIELD_TYPE_WRITEV:
    case YIELD_TYPE_SEND: case YIELD_TYPE_SENDTO: case YIELD_TYPE_SENDMSG:
        netpoller_writefd(fd, gr);
        break;
    default:
        assert(fd >= 0);
        netpoller_readfd(fd, gr);
        break;
    }

}
