
#include <event2/event.h>
#include <event2/thread.h>
#include <event2/dns.h>
#include <stdio.h>

#include "coronagc.h"
#include "coronapriv.h"
#include "futex.h"

// 由于 hook中没有hook epoll_wait, epoll_create,
// 所以在这是可以使用libev/libuv。
// 如果以后hook，则这的实现无效了。看似也不需要hook epoll

// typedef struct ev_loop ev_loop;

#define EV_IO EV_READ|EV_WRITE|EV_CLOSED
#define EV_TIMER EV_TIMEOUT
#define EV_DNS_RESOLV 0

typedef struct netpoller {
    struct event_base * loop;
    struct evdns_base* dnsbase;
    HashTable* watchers; // ev_watcher* => fiber*
    pmutex_t evmu;
} netpoller;

static netpoller* gnpl__ = 0;

static void netpoller_logcb(int severity, const char *msg) {
    // linfo("lvl=%d msg=%s\n", severity, msg);
}
void netpoller_use_threads() {
    evthread_use_pthreads();
    // or evthread_use_windows_threads()
    event_set_mem_functions(crn_gc_malloc, crn_gc_realloc, crn_gc_free);
    event_set_log_callback(netpoller_logcb);
}

netpoller* netpoller_new() {
    assert(gnpl__ == 0);
    netpoller* np = (netpoller*)crn_gc_malloc(sizeof(netpoller));
    pmutex_init(&np->evmu, nilptr);
    np->loop = event_base_new();
    assert(np->loop != 0);
    np->dnsbase = evdns_base_new(np->loop, 1);

    // hashtable_new(&np->watchers);

    gnpl__ = np;
    log_info("using backend libevent");
    return np;
}

void netpoller_loop() {
    netpoller* np = gnpl__;
    assert(np != 0);

    struct timeval tv = {0};
    tv.tv_sec =  0;
    tv.tv_usec = 586000;

    for (;;) {
        // int rv = event_base_dispatch(np->loop);
        int flags = EVLOOP_NO_EXIT_ON_EMPTY;
        // flags = 0;
        // int rv = event_base_loop(np->loop, flags);
        int rv = event_base_loopexit(np->loop, &tv);
        event_base_dispatch(np->loop);
        linfo("ohno, rv=%d\n", rv);
        // event_base_dump_events(np->loop, stdout);
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
    // evdata* d = crn_raw_malloc(sizeof(evdata));
    evdata* d = crn_gc_malloc(sizeof(evdata));
    d->evtyp = evtyp;
    d->data = data;
    // GC_register_finalizer(d, atstgc_finalizer_fn, nilptr, nilptr, nilptr);
    return d;
}
void evdata_free(evdata* d) {
    // crn_raw_free(d);
    crn_gc_free(d);
}

extern void crn_procer_resume_one(void* cbdata, int ytype, int grid, int mcid);

// common version callback, support ev_io, ev_timer
static
void netpoller_evwatcher_cb(evutil_socket_t fd, short events, void* arg) {
    evdata* d = (evdata*)arg;
    assert(d != 0);
    void* dd = d->data;
    int ytype = d->ytype;
    int grid = d->grid;
    int mcid = d->mcid;
    struct event* evt = d->evt;
    fiber *gr = dd;

    switch (d->evtyp) {
    case EV_TIMER:
        evtimer_del(d->evt);
        // linfo("evwoke ev=%d fd=%d(%d) ytype=%d=%s %p grid=%d, mcid=%d d=%p\n", events, fd, fd, ytype, yield_type_name(ytype), dd, gr->id, gr->mcid, d);
        break;
    case EV_IO:
        event_del(d->evt);
        break;
    default:
        linfo("wtf fd=%d %d %d\n", fd, d->evtyp, d->ytype);
        assert(1==2);
    }

    // linfo("before release d=%p\n", d);
    if (d->evtyp == EV_TIMER && fd != -1) {
        linfo("evwoke ev=%d fd=%d(%d) ytype=%d=%s %p grid=%d, mcid=%d d=%p\n",
              events, fd, fd, ytype, yield_type_name(ytype), dd, gr->id, gr->mcid, d);
        assert(fd == -1);
    }
    // if event_del then event_free, it crash
    // if direct event_free, it ok.
    // because non-persist event already run event_del by loop itself
    event_free(evt);
    evdata_free(d);
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
    d->tv.tv_sec = 3;
    d->tv.tv_usec = 456000;

    struct event* evt = event_new(np->loop, fd, EV_READ|EV_CLOSED, netpoller_evwatcher_cb, d);
    d->evt = evt;
    crn_pre_gclock_proc(__func__);
    int rv = event_add(evt, &d->tv);
    crn_post_gclock_proc(__func__);
    if (rv != 0) {
        // [warn] Epoll ADD(8193) on fd 18 failed. Old events were 0; read change was 1 (add); write change was 0 (none); close change was 1 (add): Bad file descriptor
        lwarn("add error %d %d %d\n", rv, fd, gr->id);
        event_free(evt);
        evdata_free(d);
        // assert(rv == 0);
        return;
    }

    if (d != nilptr) {
        linfo("event_add d=%p fd=%d ytype=%d rv=%d\n", d, fd, ytype, rv);
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
    d->tv.tv_sec = 3;
    d->tv.tv_usec = 456000;

    struct event* evt = event_new(np->loop, fd, EV_WRITE|EV_CLOSED, netpoller_evwatcher_cb, d);
    d->evt = evt;
    crn_pre_gclock_proc(__func__);
    int rv = event_add(evt, &d->tv);
    crn_post_gclock_proc(__func__);
    if (rv != 0) {
        lwarn("add error %d %d %d\n", rv, fd, gr->id);
        event_free(evt);
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

    struct event* tmer = evtimer_new(np->loop, netpoller_evwatcher_cb, d);
    // struct event* tmer = event_new(np->loop, -1, 0, netpoller_evwatcher_cb, d);
    d->evt = tmer;
    crn_pre_gclock_proc(__func__);
    int rv = evtimer_add(tmer, &d->tv);
    crn_post_gclock_proc(__func__);
    if (rv != 0) {
        lwarn("add error %d %ld %d\n", rv, ns, gr->id);
        event_free(tmer);
        evdata_free(d);
        // assert(rv == 0);
        return;
    }

    // linfo("timer add d=%p %ld ytype=%d=%s sec=%d, usec=%d\n", d, ns, ytype, yield_type_name(ytype), d->tv.tv_sec, d->tv.tv_usec);
}

static struct addrinfo* netpoller_dump_addrinfo(struct evutil_addrinfo* addr) {
    struct addrinfo *resout = nilptr;
    struct evutil_addrinfo *ai;
    int cnter = 0;
    for (ai = addr; ai; ai = ai->ai_next, cnter++) {
        char buf[128] = {0};
        const char *s = NULL;
        if (ai->ai_family == AF_INET) {
            struct sockaddr_in *sin = (struct sockaddr_in *)ai->ai_addr;
            s = evutil_inet_ntop(AF_INET, &sin->sin_addr, buf, 128);
        } else if (ai->ai_family == AF_INET6) {
            struct sockaddr_in6 *sin6 = (struct sockaddr_in6 *)ai->ai_addr;
            s = evutil_inet_ntop(AF_INET6, &sin6->sin6_addr, buf, 128);
        }
        if (s) {
            // linfo("-> %s\n", s);
        }else{
            break;
        }

        struct addrinfo* tai = crn_gc_malloc(sizeof(struct addrinfo));
        tai->ai_flags = ai->ai_flags;
        tai->ai_family = ai->ai_family;
        tai->ai_socktype = ai->ai_socktype;
        tai->ai_protocol = ai->ai_protocol;
        tai->ai_addrlen = ai->ai_addrlen;
        tai->ai_addr = crn_gc_malloc(sizeof(struct sockaddr));
        memcpy(tai->ai_addr, ai->ai_addr, sizeof(struct sockaddr));
        tai->ai_canonname = ai->ai_canonname == nilptr ? nilptr : strdup(ai->ai_canonname);
        tai->ai_next = resout;
        resout = tai;
    }
    return resout;
}
extern bool crn_procer_resume_prechk(void* gr_, int ytype, int grid, int mcid);
static
void evdns_resolv_cbproc(int errcode, struct evutil_addrinfo *addr, void *ptr)
{
    evdata* d = (evdata*)ptr;
    fiber* gr = (fiber*)d->data;
    void* dd = d->data;
    int ytype = d->ytype;
    int grid = d->grid;
    int mcid = d->mcid;
    int* ioerrcode = d->errcode;

    const char *name = (const char*)d->fd;
    struct addrinfo *resout = nilptr;
    struct addrinfo **resout2 = (struct addrinfo**)d->out;
    if (errcode) {
        assert(addr == nilptr);
        lerror("%s -> %s\n", name, evutil_gai_strerror(errcode));
    } else {
        if (addr->ai_canonname) {
            // linfo(" [%s]\n", addr->ai_canonname);
        }
        resout = netpoller_dump_addrinfo(addr);
        evutil_freeaddrinfo(addr);
    }

    evdata_free(d);

    //crn_post_gclock_proc(__func__);
    if (crn_procer_resume_prechk(dd,ytype,grid,mcid) == false) {
        lwarn("oops, seems gr gone %d\n", grid);
        if (resout != nilptr) { freeaddrinfo(resout); }
        return;
    }
    // void* chkderef = *resout2; // FIXME deref crash
    *resout2 = resout;
    *ioerrcode = errcode;
    // linfo("resolved resume %d\n", grid);
    crn_procer_resume_one(dd, ytype, grid, mcid);
}

void* netpoller_dnsresolv(const char* hostname, int ytype, fiber* gr, struct addrinfo** addr, int *errcode) {
    netpoller* np = gnpl__;
    //crn_pre_gclock_proc(__func__);

    void* chkderef = *addr;
    evdata* d = evdata_new(EV_DNS_RESOLV, gr);
    d->grid = gr->id;
    d->mcid = gr->mcid;
    d->ytype = ytype;
    d->fd = (long)hostname;
    d->out = (void**)addr;
    d->errcode = errcode;

    struct evutil_addrinfo hints;
    struct evdns_getaddrinfo_request *req;
    // struct user_data *user_data;
    memset(&hints, 0, sizeof(hints));
    hints.ai_family = AF_UNSPEC;
    hints.ai_flags = EVUTIL_AI_CANONNAME;
    /* Unless we specify a socktype, we'll get at least two entries for
     * each address: one for TCP and one for UDP. That's not what we
     * want. */
    hints.ai_socktype = SOCK_STREAM;
    hints.ai_protocol = IPPROTO_TCP;

    crn_pre_gclock_proc(__func__);
    req = evdns_getaddrinfo(np->dnsbase, hostname, NULL /* no service name given */,
                            &hints, evdns_resolv_cbproc, d);
    crn_post_gclock_proc(__func__);

    if (req == NULL) {
        lwarn("    [request for %s returned immediately]\n", hostname);
    }

    // linfo("dnsrv add d=%p %ld\n", d, hostname);
    return req;
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
