
#include <ev.h>

#include "coronapriv.h"

// 由于 hook中没有hook epoll_wait, epoll_create,
// 所以在这是可以使用libev/libuv。
// 如果以后hook，则这的实现无效了。看似也不需要hook epoll

#define HKDEBUG 1
#define linfo(fmt, ...)                                                 \
    do { if (HKDEBUG) fprintf(stderr, "%s:%d:%s ", __FILE__, __LINE__, __FUNCTION__); } while (0); \
    do { if (HKDEBUG) fprintf(stderr, fmt, __VA_ARGS__); } while (0) ;  \
    do { if (HKDEBUG) fflush(stderr); } while (0) ;

// typedef struct ev_loop ev_loop;

typedef struct netpoller {
    struct ev_loop* loop;
    HashTable* watchers; // ev_watcher* => fiber*
} netpoller;

static netpoller* gnpl__ = 0;

netpoller* netpoller_new() {
    assert(gnpl__ == 0);
    netpoller* np = (netpoller*)crn_raw_malloc(sizeof(netpoller));
    np->loop = ev_loop_new(ev_recommended_backends ());
    assert(np->loop != 0);
    ev_set_io_collect_interval(np->loop, 0.1);
    ev_set_timeout_collect_interval(np->loop, 0.1);

    hashtable_new(&np->watchers);

    gnpl__ = np;
    return np;
}

static double evtmerval(struct ev_loop* loop, double val) {
    return ev_time()-ev_now(loop)+val;
}
static void forevertmer_cb(struct ev_loop* loop, ev_timer* w, int revts) {
    // linfo("forevertmer timedout %d, active=%d, remaining=%d\n", revts, w->active, ev_timer_remaining(loop, w));
    // ev_timer_stop(loop, w);
    // ev_timer_set(w, evtmerval(loop, 3.5), evtmerval(loop, 3.5));
    // ev_timer_start(loop, w);
}

void netpoller_loop() {
    netpoller* np = gnpl__;
    assert(np != 0);

    ev_timer forevertmer;
    forevertmer.data = (void*)567;
    ev_timer_init(&forevertmer, forevertmer_cb, 0.000000001, 1); // 首次执行时间要短
    ev_timer_start(np->loop, &forevertmer);

    for (;;) {
        bool rv = ev_run(np->loop, 0);
        linfo("ohno, rv=%d\n", rv);
    }
    assert(1==2);
}

typedef struct evdata {
    int evtyp;
    void* data;
    int ytype;
} evdata;
evdata* evdata_new(int evtyp, void* data) {
    evdata* d = crn_malloc_st(evdata);
    d->evtyp = evtyp;
    d->data = data;
    return d;
}
void evdata_free(evdata* d) { crn_raw_free(d); }

extern void crn_procer_resume_some(void* cbdata);

// common version callback, support ev_io, ev_timer
static
void netpoller_evwatcher_cb(struct ev_loop* loop, ev_watcher* evw, int revents) {
    evdata* d = (evdata*)evw->data;
    assert(d != 0);

    ev_io* iow = (ev_io*)evw;
    ev_timer* tmer = (ev_timer*)evw;

    switch (d->evtyp) {
    case EV_IO:
        ev_io_stop(loop, iow);
        break;
    case EV_TIMER:
        linfo("timer cb %p, active=%d, %f\n", evw, tmer->active, ev_timer_remaining(loop, tmer));
        ev_timer_stop(loop, tmer);
        break;
    default:
        assert(1==2);
    }

    crn_procer_resume_some(d->data);
    evdata_free(d);
    crn_free(evw);
}

static
void netpoller_readfd(int fd, void* gr) {
    netpoller* np = gnpl__;
    ev_io* iow = crn_malloc_st(ev_io);
    iow->data = evdata_new(EV_IO, gr);
    ev_io_init(iow, (void(*)(struct ev_loop*, ev_io*, int))netpoller_evwatcher_cb, fd, EV_READ|EV__IOFDSET);
    ev_io_start(np->loop, iow);
}

static
void netpoller_writefd(int fd, void* gr) {
    netpoller* np = gnpl__;
    ev_io* iow = crn_malloc_st(ev_io);
    iow->data = evdata_new(EV_IO, gr);
    ev_io_init(iow, (void(*)(struct ev_loop*, ev_io*, int))netpoller_evwatcher_cb, fd, EV_WRITE|EV__IOFDSET);
    ev_io_start(np->loop, iow);
    linfo("iow started %d w=%p\n", fd, iow);
}

static
void netpoller_timer(long ns, void* gr) {
    netpoller* np = gnpl__;

    // ev_suspend(np->loop);
    // ev_resume(np->loop);

    double after = ((double)ns)/1000000000.0;
    after = evtmerval(np->loop, after);

    ev_timer* tmer = crn_malloc_st(ev_timer);
    tmer->data = evdata_new(EV_TIMER, gr);
    ev_timer_init(tmer, (void(*)(struct ev_loop*, ev_timer*, int))netpoller_evwatcher_cb, after, 0);
    ev_timer_start(np->loop, tmer);
}

// when ytype is SLEEP/USLEEP/NANOSLEEP, fd is the nanoseconds
void netpoller_yieldfd(int fd, int ytype, void* gr) {
    assert(ytype > YIELD_TYPE_NONE);
    assert(ytype < YIELD_TYPE_MAX);

    long ns = 0;
    switch (ytype) {
    case YIELD_TYPE_SLEEP:
        ns = (long)fd*1000000000;
        netpoller_timer(ns, gr);
    case YIELD_TYPE_USLEEP:
        ns = (long)fd*1000;
        netpoller_timer(ns, gr);
        break;
    case YIELD_TYPE_NANOSLEEP:
        ns = fd;
        netpoller_timer(ns, gr);
        break;
    case YIELD_TYPE_CONNECT: case YIELD_TYPE_WRITE: case YIELD_TYPE_WRITEV:
    case YIELD_TYPE_SEND: case YIELD_TYPE_SENDTO: case YIELD_TYPE_SENDMSG:
        netpoller_writefd(fd, gr);
        break;
    default:
        netpoller_readfd(fd, gr);
    }

    switch (ytype) {
    case YIELD_TYPE_SLEEP: case YIELD_TYPE_USLEEP: case YIELD_TYPE_NANOSLEEP:
        ev_now_update(gnpl__->loop);
        ev_break(gnpl__->loop, EVBREAK_CANCEL);
    }
}
