

#include <fcntl.h>
#include <unistd.h>
#include <sys/types.h>
#include <sys/socket.h>
#include <netdb.h>
#include <sys/epoll.h>
#include <sys/timerfd.h>
#include <sys/eventfd.h>

#include "coronapriv.h"

// 由于 hook中没有hook epoll_wait, epoll_create,
// 所以在这是可以使用libev/libuv/picoev。
// 如果以后hook，则这的实现无效了。看似也不需要hook epoll

#define CXEV_IO (0x1<<5)
#define CXEV_TIMER (0x1<<6)
#define CXEV_DNS_RESOLV (0x1<<7)


extern void crn_procer_resume_one(void* cbdata, int ytype, int grid, int mcid);

typedef struct evdata {
    int evtyp;
    void* data; // fiber*
    int grid;
    int mcid;
    int ytype;
    long fd; // fd or ns or hostname
    void** out; //
    int *errcode;
    long seqno;
    struct timeval tv;
    struct timeval tv2; // absolute time
    struct event* evt;
} evdata;
typedef struct evdata2 {
    int evtyp;
    evdata* dr;
    evdata* dw;
    evdata* dt;
} evdata2;

static int crn_tree_timer_cmp(const void* k1x, const void* k2x) {
    evdata* k1 = (evdata*)k1x;
    evdata* k2 = (evdata*)k2x;
    int k1us = k1->tv2.tv_sec*CRN_USEC + k1->tv2.tv_usec;
    int k2us = k2->tv2.tv_sec*CRN_USEC + k2->tv2.tv_usec;
    return -(k1us - k2us);
}
static TreeTableConf crndftttconf = {.cmp   = crn_tree_timer_cmp,
                                      .mem_alloc  = crn_gc_malloc,
                                      .mem_calloc = crn_gc_calloc,
                                      .mem_free   = crn_gc_free};
static PQueueConf crndftpqconf = {
                                   .capacity = 8,
                                   .exp_factor = 2,
                                   .cmp   = crn_tree_timer_cmp,
                                   .mem_alloc  = crn_gc_malloc,
                                   .mem_calloc = crn_gc_calloc,
                                   .mem_free   = crn_gc_free};
// thread safe poller
typedef struct netpoller {
    int epfd;
    evdata2 *evfds[999];
    long seqno;
    int tmupfd[2]; // timer add refresh notify epoll_wait pipe
    int tmupevfd;
    PQueue* timers; // evdata2* => 0
    HashTable* watchers; // ev_watcher* => evdata2
    pmutex_t evmu;
} netpoller;

static netpoller* gnpl__ = 0;

static void netpoller_logcb(int severity, const char *msg) {
    linfo("lvl=%d msg=%s\n", severity, msg);
}
void netpoller_use_threads() {

}

netpoller* netpoller_new() {
    assert(gnpl__ == 0);
    netpoller* np = (netpoller*)crn_gc_malloc(sizeof(netpoller));
    np->epfd = epoll_create1(EPOLL_CLOEXEC);
    assert(np->epfd > 0);
    // np->dnsbase = evdns_base_new(np->loop, 1);

    np->seqno = 10000;
    int rv = pqueue_new_conf(&crndftpqconf, &np->timers);
    assert(rv == CC_OK);
    rv = pipe2(np->tmupfd, O_NONBLOCK);
    assert(rv == 0);
    np->tmupevfd = eventfd(0, EFD_CLOEXEC|EFD_NONBLOCK);
    assert(np->tmupevfd > 0);
    // hashtable_new(&np->watchers);

    gnpl__ = np;
    return np;
}

// 1/1000 秒, used in epoll_wait
static int netpoller_next_timeout() {
    netpoller* np = gnpl__;
    assert(np != 0);

    evdata* curd = nilptr;
    int tabsz = 0;
    int rv = CC_ERR_OUT_OF_RANGE;
    pthread_mutex_lock(&np->evmu);
    tabsz = pqueue_size(np->timers);
    if (tabsz > 0) {
        rv = pqueue_top(np->timers, (void**)&curd);
    }
    pthread_mutex_unlock(&np->evmu);
    assert(rv == CC_OK || rv == CC_ERR_OUT_OF_RANGE);
    if (rv == CC_ERR_OUT_OF_RANGE) {
        return -1;
    }
    evdata nowd = {0};
    gettimeofday(&nowd.tv2, nilptr);
    int diffus = -crn_tree_timer_cmp(curd, &nowd);
    // linfo("next timeout %d %d\n", diffus, diffus/1000);
    if (diffus < 0) { return 0; }
    return (int)(diffus/1000);
}

static int netpoller_resume_one(evdata* d) {
    void* dd = d->data;
    int ytype = d->ytype;
    int grid = d->grid;
    int mcid = d->mcid;
    evdata_free(d);
    // evdata2_free(d2);

    crn_procer_resume_one(dd, ytype, grid, mcid);
}

static int netpoller_dispatch_timers2() {
    netpoller* np = gnpl__;
    assert(np != 0);

    evdata nowd = {0};
    gettimeofday(&nowd.tv2, nilptr);
    evdata* expires[128] = {0};
    int expcnt = 0;

    int tabsz = 0;
    int rv = 0;
    evdata* curd = nilptr;
    pthread_mutex_lock(&np->evmu);
    tabsz = pqueue_size(np->timers);
    for (int i = 0; i < 128 && i < tabsz; i++) {
        rv = pqueue_top(np->timers, (void**)&curd);
        if (rv != CC_OK) break;
        rv = -crn_tree_timer_cmp(curd, &nowd);
        // linfo("expire i=%d/%d %p diff=%d\n", i, expcnt, curd, rv/1000);
        if ( rv <= 0) {
            expires[expcnt++] = curd;
            pqueue_pop(np->timers, nilptr);
        }else{
            break;
        }
    }
    pthread_mutex_unlock(&np->evmu);
    if (expcnt > 0) {
        curd = expires[0];
        rv = -crn_tree_timer_cmp(curd, &nowd);
        // linfo("expire cnt=%d tot=%d diffms=%d seqno=%d\n", expcnt, tabsz, rv/1000, curd->seqno);
    }
    for (int i = 0; i < expcnt; i++) {
        curd = expires[i];
        // linfo("expire i=%d/%d %p\n", i, expcnt, curd);
        netpoller_resume_one(curd);
    }
    return expcnt;
}
static int netpoller_dispatch_timers() {
    while (1) {
        int rv = netpoller_dispatch_timers2();
        if (rv == 0) break;
    }
}
static void netpoller_dispatch_fd(struct epoll_event *revt) {

}
void netpoller_loop() {
    netpoller* np = gnpl__;
    assert(np != 0);

    struct epoll_event evt = {0};
    evt.events = EPOLLIN;
    evt.data.fd = np->tmupevfd;
    int rv = epoll_ctl(np->epfd, EPOLL_CTL_ADD, np->tmupevfd, &evt);
    assert(rv == 0);

    for (;;) {
        struct epoll_event revt = {0};
        int timeout = netpoller_next_timeout();
        timeout = timeout < 0 ? CRN_MSEC*10 : timeout;
        if (timeout == 0) {
            // netpoller_dispatch_timers();
            // continue;
        }
        timeout = timeout <= 31 ? 31 : timeout;
        // linfo("next timeout %d %d\n", timeout, timeout/1000);
        rv = epoll_wait(np->epfd, &revt, 1, timeout);
        int eno = errno;
        if (rv < 0) {
            if (eno == EINTR) {
                continue;
            }
            linfo("wtf %d %s\n", rv, strerror(errno));
            assert(1==2);
        }
        assert(rv >= 0);
        if (rv == 0) {
            netpoller_dispatch_timers();
            continue;
        }
        assert(rv == 1);
        int evfd = revt.data.fd;
        int evts = revt.events;
        // linfo("gotevt %d %d\n", evfd, evts);
        if (evfd == np->tmupevfd) {
            uint64_t val;
            while(1) {
                rv = read(np->tmupevfd, &val, sizeof(uint64_t));
                if (rv <= 0) {break;}
            }
            netpoller_dispatch_timers();
            continue;
        }
        rv = epoll_ctl(np->epfd, EPOLL_CTL_DEL, evfd, nilptr);
        // assert(rv == 0);
        pthread_mutex_lock(&np->evmu);
        evdata2* d2 = np->evfds[evfd];
        if (d2 != 0) {
            np->evfds[evfd] = nilptr; // clear
            // linfo("clear fd %d\n", evfd);
        }
        pthread_mutex_unlock(&np->evmu);
        if (d2 == 0) {
            linfo("wtf, fd not found %d\n", evfd);
        }else{
            extern void netpoller_picoev_globcb(int epfd, int fd, int events, void* cbarg);
            netpoller_picoev_globcb(np->epfd, evfd, evts, d2);
        }
        netpoller_dispatch_timers();

        if (0) {
            linfo("ohno, rv=%d\n", rv);
        }
    }
    assert(1==2);
}


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

    // netpoller* np = gnpl__;
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
    // netpoller* np = gnpl__;
    assert(evtyp >= CXEV_IO && evtyp <= CXEV_DNS_RESOLV);
    evdata2* d2 = crn_gc_malloc(sizeof(evdata2));
    d2->evtyp = evtyp;
    d2->dr = d2->dw = d2->dt = 0;
    return d2;
}



// common version callback, support ev_io, ev_timer
void netpoller_picoev_globcb(int epfd, int fd, int events, void* cbarg) {
    netpoller* np = gnpl__;

    // linfo("fd=%d events=%d cbarg=%p %p\n", fd, events, cbarg, 0);
    evdata2* d2 = (evdata2*)cbarg;
    assert(d2 != 0);
    evdata* dv[3] = {0};
    int dvcnt = 0;

    int newev = 0;
    int rv = 0;
    switch (d2->evtyp) {
    case CXEV_TIMER:
        // linfo("fd=%d events=%d cbarg=%p %p\n", fd, events, cbarg, 0);
        close(fd);
        dv[dvcnt++] = d2->dt;
        d2->dt = nilptr;
        break;
    case CXEV_IO:
        if (events & EPOLLIN) {
            dv[dvcnt++] = d2->dr;
            d2->dr = nilptr;
        }
        if (events & EPOLLOUT) {
            dv[dvcnt++] = d2->dw;
            d2->dw = nilptr;
        }
        if ((events & EPOLLIN) == 0 && (events & EPOLLOUT) == 0) {
            // assert(1==2);
            linfo("woo, close??? %d\n", fd);
        }
        if (d2->dr != nilptr) { newev |= EPOLLIN; }
        if (d2->dw != nilptr) { newev |= EPOLLOUT; }
        if (newev == 0) {
        }else{
            pthread_mutex_lock(&np->evmu);
            np->evfds[fd] = d2;
            pthread_mutex_unlock(&np->evmu);
            struct epoll_event evt = {0};
            evt.events = newev | EPOLLET;
            evt.data.fd = fd;
            rv = epoll_ctl(np->epfd, EPOLL_CTL_ADD, fd, &evt);
        }
        break;
    default:
        linfo("wtf fd=%d %d %d %d\n", fd, CXEV_IO, CXEV_TIMER, CXEV_DNS_RESOLV);
        // linfo("wtf fd=%d r=%d w=%d\n", fd, events&PICOEV_READ, events&PICOEV_WRITE);
        linfo("wtf fd=%d %p %d %p %p %p\n", fd, d2, d2->evtyp, d2->dr, d2->dw, d2->dt);
        assert(1==2);
    }
    assert(dvcnt > 0);
    for (int i = 0; i < 3; i++) {
        evdata* d = dv[i];
        if (d == nilptr) { break; }
        netpoller_resume_one(d);
    }
}

extern void crn_pre_gclock_proc(const char* funcname);
extern void crn_post_gclock_proc(const char* funcname);

static
void netpoller_readfd(int fd, int ytype, fiber* gr) {
    netpoller* np = gnpl__;
    evdata* d = evdata_new(CXEV_IO, gr);
    d->grid = gr->id;
    d->mcid = gr->mcid;
    d->ytype = ytype;
    d->fd = fd;

    crn_pre_gclock_proc(__func__);
    int rv = 0;
    pthread_mutex_lock(&np->evmu);
    int inuse = np->evfds[fd] != 0;
    // assert (inuse == 0);
    if (inuse) {
        evdata2* d2 = np->evfds[fd];
        int druse = d2->dr != nilptr;
        int samefib = druse ? d2->dr->data == gr : 0;
        int override = 0;
        if (druse && samefib) {
            // ignore ok?
        }else{
            override = 1;
            d2->dr = d;
            int newev = EPOLLET;
            if (d2->dw != nilptr) { newev |= EPOLLOUT; }
            newev |= EPOLLIN;
            rv = epoll_ctl(np->epfd, EPOLL_CTL_DEL, fd, 0);
            // assert(rv == 0);
            struct epoll_event evt = {0};
            evt.events = newev;
            evt.data.fd = fd;
            rv = epoll_ctl(np->epfd, EPOLL_CTL_ADD, fd, &evt);
        }
        if (druse && samefib && !override) {
            // ignored operation
        }else{
            linfo("add r reset %d druse=%d samefib=%d override=%d\n", fd, druse, samefib, override);
        }
    }else{
        evdata2* d2 = evdata2_new(d->evtyp);
        d2->dr = d;
        struct epoll_event evt = {0};
        evt.events = EPOLLIN | EPOLLET;
        evt.data.fd = fd;
        np->evfds[fd] = d2;
        rv = epoll_ctl(np->epfd, EPOLL_CTL_ADD, fd, &evt);
        // linfo("add r new %d\n", fd);
    }
    pthread_mutex_unlock(&np->evmu);
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
    evdata* d = evdata_new(CXEV_IO, gr);
    d->grid = gr->id;
    d->mcid = gr->mcid;
    d->ytype = ytype;
    d->fd = fd;

    int rv = 0;
    crn_pre_gclock_proc(__func__);
    pthread_mutex_lock(&np->evmu);
    int inuse = np->evfds[fd] != 0;
    if (inuse) {
        evdata2* d2 = np->evfds[fd];
        int dwuse = d2->dw != nilptr;
        int samefib = dwuse ? d2->dw->data == gr : 0;
        int override = 0;
        if (dwuse && samefib) {
        }else{
            override = 1;
            d2->dw = d;
            int newev = EPOLLET;
            if (d2->dr != nilptr) { newev |= EPOLLIN; }
            newev |= EPOLLOUT;
            rv = epoll_ctl(np->epfd, EPOLL_CTL_DEL, fd, 0);
            // assert(rv == 0);
            struct epoll_event evt = {0};
            evt.events = newev;
            evt.data.fd = fd;
            rv = epoll_ctl(np->epfd, EPOLL_CTL_ADD, fd, &evt);
        }
        if (dwuse && samefib && !override) {
            // ignored operation
        }else{
            linfo("add w reset %d dwuse=%d samefib=%d override=%d\n", fd, dwuse, samefib, override);
        }
    }else{
        evdata2* d2 = evdata2_new(d->evtyp);
        d2->dw = d;
        struct epoll_event evt = {0};
        evt.events = EPOLLOUT | EPOLLET;
        evt.data.fd = fd;
        np->evfds[fd] = d2;
        rv = epoll_ctl(np->epfd, EPOLL_CTL_ADD, fd, &evt);
        // linfo("add w new %d\n", fd);
    }
    pthread_mutex_unlock(&np->evmu);
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

    evdata* d = evdata_new(CXEV_TIMER, gr);
    d->grid = gr->id;
    d->mcid = gr->mcid;
    d->ytype = ytype;
    d->fd = ns;
    d->tv.tv_sec = ns/1000000000;
    d->tv.tv_usec = ns/1000 % 1000000;
    gettimeofday(&d->tv2, nilptr);
    int usec = d->tv2.tv_usec + d->tv.tv_usec;
    d->tv2.tv_sec += d->tv.tv_sec + usec/1000000;
    d->tv2.tv_usec = usec%1000000;

    int rv = 0;
    crn_pre_gclock_proc(__func__);
    pthread_mutex_lock(&np->evmu);
        d->seqno = ++np->seqno;
        // rv = treetable_add(np->timers, d, (void*)0);
        rv = pqueue_push(np->timers, d);
        assert(rv == CC_OK);
    pthread_mutex_unlock(&np->evmu);
    crn_post_gclock_proc(__func__);
    uint64_t tmval = 1;
    rv = write(np->tmupevfd, &tmval, sizeof(uint64_t));
    rv = rv == sizeof(uint64_t) ? 0 : rv;
    if (rv != 0) {
        lwarn("add error %d %ld %d\n", rv, ns, gr->id);
        // evdata2_free(tmer);
        evdata_free(d);
        // assert(rv == 0);
        return;
    }

    // linfo("timer add %d d=%p %ld sec=%d nsec=%d\n", tmfd, d, ns, ts.tv_sec, ts.tv_nsec);
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
