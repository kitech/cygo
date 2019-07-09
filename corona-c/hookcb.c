
#include <stdio.h>
#include <stdbool.h>
#include <assert.h>
#include <fcntl.h>
#include <sys/socket.h>

// #include <gc/gc.h>
#include "collectc/hashtable.h"
#include "collectc/array.h"

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
} fdcontext;

typedef struct hookcb {
    HashTable* fdctxs; // fd => fdcontext*
    pmutex_t mu;
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
    GC_REGISTER_FINALIZER(fdctx, 0, 0, 0, 0);
    int fd = fdctx->fd;
    void* optr = fdctx;
    crn_gc_free(fdctx);
    // linfo("fdctx freed %d %p\n", fd, optr);
    assert(fd >= 0 && fd < 60000);
}

typedef int(*fcntl_t)(int __fd, int __cmd, ...);
extern fcntl_t fcntl_f;

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

    HashTableConf htconf;
    hashtable_conf_init(&htconf);
    htconf.hash = hashtable_hash_ptr;
    htconf.key_compare = hashtable_cmp_int;
    htconf.key_length = sizeof(void*);
    htconf.mem_alloc = crn_gc_malloc;
    htconf.mem_free = crn_gc_free;
    htconf.mem_calloc = crn_gc_calloc;

    hashtable_new_conf(&htconf, &hkcb->fdctxs);
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
    pmutex_lock(&hkcb->mu);
    int rv = hashtable_remove(hkcb->fdctxs, (void*)(uintptr_t)fd, (void**)&oldfdctx);
    rv = hashtable_add(hkcb->fdctxs, (void*)(uintptr_t)fd, (void*)fdctx);
    pmutex_unlock(&hkcb->mu);
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
    pmutex_lock(&hkcb->mu);
    int rv = hashtable_remove(hkcb->fdctxs, (void*)(uintptr_t)fd, (void**)&fdctx);
    pmutex_unlock(&hkcb->mu);
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
    pmutex_lock(&hkcb->mu);
    int rv = hashtable_get(hkcb->fdctxs, (void*)(uintptr_t)from, (void**)&fdctx);
    pmutex_unlock(&hkcb->mu);
    assert(rv == CC_OK);
    assert(fdctx != 0);

    fdcontext* tofdctx = fdcontext_new(to);
    memcpy(tofdctx, fdctx, sizeof(fdcontext));
    tofdctx->fd = to;
    pmutex_lock(&hkcb->mu);
    rv = hashtable_add(hkcb->fdctxs, (void*)(uintptr_t)to, tofdctx);
    pmutex_unlock(&hkcb->mu);
    assert(rv == CC_OK);
}

fdcontext* hookcb_get_fdcontext(int fd) {
    hookcb* hkcb = hookcb_get();
    if (hkcb == 0) return 0;

    fdcontext* fdctx = 0;
    pmutex_lock(&hkcb->mu);
    int rv = hashtable_get(hkcb->fdctxs, (void*)(uintptr_t)fd, (void**)&fdctx);
    pmutex_unlock(&hkcb->mu);
    if (fdctx == 0) {
        // assert(fdctx != 0);
    }
    return fdctx;
}
