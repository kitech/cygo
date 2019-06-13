
#include <stdio.h>
#include <stdbool.h>
#include <assert.h>
#include <fcntl.h>
#include <sys/socket.h>

// #include <gc/gc.h>
#include "collectc/hashtable.h"
#include "collectc/array.h"

#include "noropriv.h"
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
} fdcontext;

typedef struct hookcb {
    HashTable* fdctxs; // fd => fdcontext*
    mtx_t mu;
} hookcb;

fdcontext* fdcontext_new(int fd) {
    fdcontext* fdctx = (fdcontext*)malloc(sizeof(fdcontext));
    fdctx->fd = fd;
    return fdctx;
}

typedef int(*fcntl_t)(int __fd, int __cmd, ...);
extern fcntl_t fcntl_f;
bool fd_is_nonblocking(int fd) {
    int flags = fcntl_f(fd, F_GETFL, 0);
    bool old = flags & O_NONBLOCK;
    return old;
}
int fdcontext_set_nonblocking(fdcontext*fdctx, bool isNonBlocking) {
    int fd = fdctx->fd;
    int flags = fcntl_f(fd, F_GETFL, 0);
    bool old = flags & O_NONBLOCK;
    if (isNonBlocking == old)  return old;

    int rv = fcntl_f(fd, F_SETFL,
            isNonBlocking ? (flags | O_NONBLOCK) : (flags & ~O_NONBLOCK));
    return rv;
}
bool fdcontext_is_socket(fdcontext*fdctx) {return fdctx->fdty == FDISSOCKET; }
bool fdcontext_is_tcpsocket(fdcontext*fdctx) {
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

hookcb* hookcb_new() {
    // so, this is live forever, not use GC_malloc
    hookcb* hkcb = (hookcb*)calloc(1, sizeof(hookcb));
    HashTableConf htconf;
    hashtable_conf_init(&htconf);
    htconf.hash = hashtable_hash_ptr;
    htconf.key_compare = hashtable_cmp_int;
    htconf.key_length = sizeof(void*);
    hashtable_new_conf(&htconf, &hkcb->fdctxs);

    return hkcb;
}

hookcb* hookcb_get() {
    if (ghkcb__ == 0) {
        hookcb* hkcb = hookcb_new();
        ghkcb__ = hkcb;
    }
    assert (ghkcb__ != 0);
    return ghkcb__;
}

void hookcb_oncreate(int fd, int fdty, bool isNonBlocking, int domain, int sockty, int protocol) {
    hookcb* hkcb = hookcb_get();
    if (hkcb == 0) return ;
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

    if (noro_in_processor() && fdty == FDISSOCKET)
    if (!fd_is_nonblocking(fd)) {
        int rv = fdcontext_set_nonblocking(fdctx, true);
        assert(fd_is_nonblocking(fd) == true);
    }
    mtx_lock(&hkcb->mu);
    hashtable_add(hkcb->fdctxs, (void*)(uintptr_t)fd, (void*)fdctx);
    mtx_unlock(&hkcb->mu);
}

void hookcb_onclose(int fd) {
    hookcb* hkcb = hookcb_get();
    if (hkcb == 0) return ;
    // linfo("fd closed %d\n", fd);

    fdcontext* fdctx = 0;
    mtx_lock(&hkcb->mu);
    hashtable_remove(hkcb->fdctxs, (void*)(uintptr_t)fd, (void**)&fdctx);
    mtx_unlock(&hkcb->mu);
    // maybe not found when just startup
    if (fdctx == 0) {
        linfo("fd not found in context %d\n", fd);
    }
}

void hookcb_ondup(int from, int to) {
    hookcb* hkcb = hookcb_get();
    if (hkcb == 0) return ;

    fdcontext* fdctx = 0;
    mtx_lock(&hkcb->mu);
    hashtable_get(hkcb->fdctxs, (void*)(uintptr_t)from, (void**)&fdctx);
    mtx_unlock(&hkcb->mu);
    assert(fdctx != 0);
    fdcontext* tofdctx = fdcontext_new(to);
    memcpy(tofdctx, fdctx, sizeof(fdcontext));
    tofdctx->fd = to;
}

fdcontext* hookcb_get_fdcontext(int fd) {
    hookcb* hkcb = hookcb_get();
    if (hkcb == 0) return 0;

    fdcontext* fdctx = 0;
    mtx_lock(&hkcb->mu);
    hashtable_get(hkcb->fdctxs, (void*)(uintptr_t)fd, (void**)&fdctx);
    mtx_unlock(&hkcb->mu);
    assert(fdctx != 0);
    return fdctx;
}
