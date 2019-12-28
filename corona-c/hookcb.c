
#include <stdio.h>
#include <stdbool.h>
#include <assert.h>
#include <fcntl.h>
#include <sys/socket.h>

#include "coronapriv.h"
#include "hookcb.h"

typedef struct fdcontext {
    int fd;
    int fdty; // socket, pipe
    bool isNonBlocking;
    int tcpconntimeo;
    int recvtimeo;
    int sendtimeo;

    // attribute
    int domain;
    int sockty; // tcp/udp...
    int protocol; //
    time_t tm;
    bool inpoll; // in select/poll
} fdcontext;

typedef struct hookcb {
    crnmap *fdctxs;
} hookcb;

static void fdcontext_finalizer(void* ptr) {
    fdcontext* fdctx = (fdcontext*)ptr;
    time_t nowt = time(0);
    linfo("fdctx dtor %p %d %ld\n", ptr, fdctx->fd, nowt - fdctx->tm);
    // assert(1==2);
}
fdcontext* fdcontext_new(int fd) {
    fdcontext* fdctx = (fdcontext*)crn_gc_malloc(sizeof(fdcontext));
    fdctx->fd = fd;
    fdctx->tm = time(0);
    assert(fd != 0);
    crn_set_finalizer(fdctx,fdcontext_finalizer);
    return fdctx;
}
void fdcontext_free(fdcontext* fdctx) {
    crn_set_finalizer(fdctx,nilptr);
    int fd = fdctx->fd;
    void* optr = fdctx;
    crn_gc_free(fdctx);
    // linfo("fdctx freed %d %p\n", fd, optr);
    assert(fd >= 0 && fd < 60000);
}

typedef int(*fcntl_t)(int __fd, int __cmd, ...);
extern fcntl_t fcntl_f;
typedef int (*setsockopt_t)(int sockfd, int level, int optname,
                            const void *optval, socklen_t optlen);
extern setsockopt_t setsockopt_f;


bool fd_is_nonblocking(int fd) {
    int flags = fcntl_f(fd, F_GETFL, 0);
    bool old = flags & O_NONBLOCK;
    return old;
}
int fdcontext_set_nonblocking(fdcontext*fdctx, bool isNonBlocking) {
    if (fdctx == 0) {
        return 0;
    }

    int fd = fdctx->fd;
    int flags = fcntl_f(fd, F_GETFL, 0);
    bool old = flags & O_NONBLOCK;
    if (isNonBlocking == old)  return old;

    int rv = fcntl_f(fd, F_SETFL,
            isNonBlocking ? (flags | O_NONBLOCK) : (flags & ~O_NONBLOCK));
    {
        int one = 1;
        // setsockopt(fd, SOL_SOCKET, SO_REUSEADDR, &one, sizeof(int));
        // setsockopt(fd, IPPROTO_TCP, TCP_NODELAY, &one, sizeof(int));
    }
    return rv;
}
int hookcb_fd_set_nonblocking(int fd, bool isNonBlocking) {
    fdcontext* fdctx = hookcb_get_fdcontext(fd);
    if (fdctx == 0) {
        ldebug("fdctx nil %d, %d\n", fd, isNonBlocking);
        return 0;
    }
    return fdcontext_set_nonblocking(fdctx, isNonBlocking);
}
bool fdcontext_is_socket(fdcontext*fdctx) {
    if (fdctx == 0) return false;
    return fdctx->fdty == FDISSOCKET;
}
bool fdcontext_is_tcpsocket(fdcontext*fdctx) {
    if (fdctx == 0) return false;
    return fdctx->fdty == FDISSOCKET && fdctx->sockty == SOCK_STREAM;
}
bool fdcontext_is_nonblocking(fdcontext*fdctx){ return fdctx->isNonBlocking; }

// global static vars
static hookcb* ghkcb__ = {0};

static int hashtable_cmp_int(const void *key1, const void *key2) {
    if (key1 == key2) return 0;
    else if((uintptr_t)(key1) > (uintptr_t)(key2)) return 1;
    else return -1;
}

static void hookcb_finalizer(void* ptr) {
    linfo("hkcb dtor %p\n", ptr);
}
static void hookcbht_finalizer(void* ptr) {
    linfo("hkcbht dtor %p\n", ptr);
    assert(1==2);
}
hookcb* hookcb_new() {
    // so, this is live forever, not use GC_malloc
    hookcb* hkcb = (hookcb*)crn_gc_malloc(sizeof(hookcb));
    crn_set_finalizer(hkcb->fdctxs, hookcb_finalizer);

    hkcb->fdctxs = crnmap_new_uintptr();
    crn_set_finalizer(hkcb->fdctxs, hookcbht_finalizer);

    return hkcb;
}

extern bool gcinited;
static pmutex_t hkcbgetmu;
hookcb* hookcb_get() {
    assert(gcinited == true);
    if (ghkcb__ == 0) {
        pmutex_lock(&hkcbgetmu);
        if (ghkcb__ == 0) {
            hookcb* hkcb = hookcb_new();
            assert(ghkcb__ == 0);
            ghkcb__ = hkcb;
        }
        pmutex_unlock(&hkcbgetmu);
    }
    assert (ghkcb__ != 0);
    return ghkcb__;
}


void hookcb_oncreate(int fd, int fdty, bool isNonBlocking, int domain, int sockty, int protocol) {
    if (!crn_in_procer()) return;
    if (!gcinited) return;
    hookcb* hkcb = hookcb_get();
    if (hkcb == 0) {
        linfo("why nil %d\n", fd);
        return ;
    }
    if (!fd_is_nonblocking(fd)) {
        // set nonblocking???
        // linfo("fd is blocking %d, nb=%d\n", fd, fd_is_nonblocking(fd));
    }
    bool yes = true; // MSG_NOSIGNAL for linux?, SO_NOSIGPIPE for BSD/MAC
    // int ret = setsockopt_f(fd, SOL_SOCKET, SO_NOSIGPIPE, &yes, sizeof(yes));

    fdcontext* fdctx = fdcontext_new(fd);
    fdctx->fdty = fdty;
    fdctx->isNonBlocking = isNonBlocking;
    fdctx->domain = domain;
    fdctx->sockty = sockty;
    fdctx->protocol = protocol;
    // linfo("fdctx new %d %p\n", fd, fdctx);

    if (crn_in_procer() && fdty == FDISSOCKET)
    if (!fd_is_nonblocking(fd)) {
        int rv = fdcontext_set_nonblocking(fdctx, true);
        assert(fd_is_nonblocking(fd) == true);
    }

    fdcontext* oldfdctx = 0;
    int rv = crnmap_remove(hkcb->fdctxs,(uintptr_t)fd,(void**)&oldfdctx);
    rv = crnmap_add(hkcb->fdctxs,(uintptr_t)fd,(void*)fdctx);
    assert(rv == CC_OK);
    if (oldfdctx != nilptr) {
        fdcontext_free(oldfdctx);
    }
}

void hookcb_onclose(int fd) {
    if (!crn_in_procer()) return;
    if (!gcinited) return;
    hookcb* hkcb = hookcb_get();
    if (hkcb == 0) return ;
    // linfo("fd closed %d\n", fd);

    fdcontext* fdctx = 0;
    int rv = crnmap_remove(hkcb->fdctxs,(uintptr_t)fd,(void**)&fdctx);
    // maybe not found when just startup
    if (fdctx == 0) {
        linfo("fd not found in context %d\n", fd);
    }else{
        fdcontext_free(fdctx);
    }
}

void hookcb_ondup(int from, int to) {
    hookcb* hkcb = hookcb_get();
    if (hkcb == 0) return ;

    fdcontext* fdctx = 0;
    int rv = crnmap_get(hkcb->fdctxs,(uintptr_t)from,(void**)&fdctx);
    assert(rv == CC_OK);
    assert(fdctx != 0);

    fdcontext* tofdctx = fdcontext_new(to);
    memcpy(tofdctx, fdctx, sizeof(fdcontext));
    tofdctx->fd = to;
    rv = crnmap_add(hkcb->fdctxs,(uintptr_t)to,tofdctx);
    assert(rv == CC_OK);
}

//
void hookcb_setin_poll(int fd, bool set) {
    hookcb* hkcb = hookcb_get();
    if (hkcb == 0) return ;

    fdcontext* fdctx = 0;
    int rv = crnmap_get(hkcb->fdctxs,(uintptr_t)fd,(void**)&fdctx);
    assert(rv == CC_OK);
    assert(fdctx != 0);

    fdctx->inpoll = set;
}
bool hookcb_getin_poll(int fd) {
    hookcb* hkcb = hookcb_get();
    if (hkcb == 0) return ;

    fdcontext* fdctx = 0;
    int rv = crnmap_get(hkcb->fdctxs,(uintptr_t)fd,(void**)&fdctx);
    assert(rv == CC_OK);
    assert(fdctx != 0);

    return fdctx->inpoll;
}

fdcontext* hookcb_get_fdcontext(int fd) {
    hookcb* hkcb = hookcb_get();
    if (hkcb == 0) return 0;

    fdcontext* fdctx = 0;
    int rv = crnmap_get(hkcb->fdctxs,(uintptr_t)fd,(void**)&fdctx);
    if (fdctx == 0) {
        // assert(fdctx != 0);
    }
    return fdctx;
}

/*

  int rcvbuf = 0;
  int valen = sizeof(int);
  getsockopt(fds[i].fd, SOL_SOCKET, SO_RCVBUF, &rcvbuf, &valen);
  int sndbuf = 0;
  int valen2 = sizeof(int);
  getsockopt(fds[i].fd, SOL_SOCKET, SO_SNDBUF, &sndbuf, &valen2);
  linfo("fd=%d evs=%d POLLIN=%d POLLOUT=%d RCVBUF=%d SNDBUF=%d\n",
  fds[i].fd, fds[i].events, POLLIN, POLLOUT,rcvbuf, sndbuf);
  rcvbuf = 1000;
  setsockopt(fds[i].fd, SOL_SOCKET, SO_RCVBUF, &rcvbuf, sizeof(int));
  sndbuf = 5000;
  setsockopt(fds[i].fd, SOL_SOCKET, SO_SNDBUF, &sndbuf, sizeof(int));

*/
