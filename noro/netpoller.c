#include "noropriv.h"

#include <ev.h>

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
    HashTable* watchers; // ev_watcher* => goroutine*
    // HashTable* fds; // fd => ev_watcher*
} netpoller;

static netpoller* gnpl__ = 0;

netpoller* netpoller_new() {
    assert(gnpl__ == 0);
    netpoller* np = (netpoller*)calloc(1, sizeof(netpoller));
    np->loop = ev_loop_new(0);
    assert(np->loop != 0);

    hashtable_new(&np->watchers);

    gnpl__ = np;
    return np;
}

static void forevertmer_cb(struct ev_loop* loop, ev_timer* w, int revts) {
    // linfo("forevertmer timedout %d\n", revts);
    ev_timer_stop(loop, w);
    ev_timer_set(w, 5.5, 0);
    ev_timer_start(loop, w);
}

void netpoller_loop() {
    netpoller* np = gnpl__;
    assert(np != 0);

    ev_timer forevertmer;
    forevertmer.data = (void*)567;
    ev_timer_init(&forevertmer, forevertmer_cb, 5.5, 1);
    ev_timer_start(np->loop, &forevertmer);

    for (;;) {
        bool rv = ev_run(np->loop, 0);
        linfo("ohno, rv=%d\n", rv);
    }
    assert(1==2);
}

typedef struct evdata {
    int typ;
    void* data;
} evdata;
evdata* evdata_new(int typ, void* data) {
    evdata* d = noro_malloc_st(evdata);
    d->typ = EV_IO;
    d->data = data;
    return d;
}
void evdata_free(evdata* d) { noro_free(d); }

extern void noro_processor_resume_some(void* cbdata);

// common version callback, support ev_io, ev_timer
static
void netpoller_evwatcher_cb(struct ev_loop* loop, ev_watcher* evw, int revents) {
    evdata* d = (evdata*)evw->data;
    assert(d != 0);

    ev_io* iow = (ev_io*)evw;
    ev_timer* tmer = (ev_timer*)evw;

    switch (d->typ) {
    case EV_IO:
        ev_io_stop(loop, iow);
        break;
    case EV_TIMER:
        ev_timer_stop(loop, tmer);
        break;
    default:
        assert(1==2);
    }

    noro_processor_resume_some(d->data);
    evdata_free(d);
    noro_free(evw);
}

void netpoller_timer(long ns, void* gr) {
    netpoller* np = gnpl__;
    ev_timer* tmer = noro_malloc_st(ev_timer);
    tmer->data = evdata_new(EV_TIMER, gr);
    ev_timer_init(tmer, netpoller_evwatcher_cb, ns, 0);
    ev_timer_start(np->loop, tmer);
}

static
void netpoller_addfd(int fd, void* gr) {

}
static
void netpoller_delfd(int fd, void* gr) {

}
static
ev_watcher* netpoller_watcher_getoradd(int fd, void* gr) {
    netpoller* np = gnpl__;
}

void netpoller_readfd(int fd, void* gr) {
    netpoller* np = gnpl__;
    ev_io* iow = noro_malloc_st(ev_io);
    iow->data = evdata_new(EV_IO, gr);
    ev_io_init(iow, netpoller_evwatcher_cb, fd, EV_READ|EV__IOFDSET);
    ev_io_start(np->loop, iow);
    linfo("ior started %d w=%p\n", fd, iow);
}

void netpoller_writefd(int fd, void* gr) {
    netpoller* np = gnpl__;
    ev_io* iow = noro_malloc_st(ev_io);
    iow->data = evdata_new(EV_IO, gr);
    ev_io_init(iow, netpoller_evwatcher_cb, fd, EV_WRITE|EV__IOFDSET);
    ev_io_start(np->loop, iow);
    linfo("iow started %d w=%p\n", fd, iow);
}

