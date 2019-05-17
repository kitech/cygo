#include <unistd.h>
#include <stdio.h>
#include <assert.h>

#include <gc/gc.h>
#include <coro.h>
#include <collectc/hashtable.h>
#include <collectc/array.h>
#include "noro.h"

#define HKDEBUG 1
#define linfo(fmt, ...)                                                 \
    do { if (HKDEBUG) fprintf(stderr, "%s:%d:%s ", __FILE__, __LINE__, __FUNCTION__); } while (0); \
    do { if (HKDEBUG) fprintf(stderr, fmt, __VA_ARGS__); } while (0) ;


#include <sys/epoll.h>
#include "hook.h"
extern fcntl_t fcntl_f;
extern getsockopt_t getsockopt_f;
extern setsockopt_t setsockopt_f;
extern epoll_wait_t epoll_wait_f;

int noro_epoll_create() {
    int fd = epoll_create1(EPOLL_CLOEXEC);
    return fd;
}
int noro_epoll_wait(int epfd, struct epoll_event *events,
                     int maxevents, int timeout) {
    return epoll_wait_f(epfd, events, maxevents, timeout);
}

#include <unistd.h>
#include <sys/syscall.h>

pid_t gettid() {
#ifdef SYS_gettid
    pid_t tid = syscall(SYS_gettid);
    return tid;
#else
#error "SYS_gettid unavailable on this system"
    return 0;
#endif
}

#include <noro.h>
#include <coro.h>

typedef struct coro_stack coro_stack;

///
extern void corowp_create(coro_context *ctx, coro_func coro, void *arg, void *sptr,  size_t ssze);
extern void corowp_transfer(coro_context *prev, coro_context *next);
extern void corowp_destroy (coro_context *ctx);
extern int corowp_stack_alloc (coro_stack *stack, unsigned int size);
extern void corowp_stack_free(coro_stack* stack);



// 每个goroutine同时只能属于某一个machine
typedef struct goroutine {
    int id;
    coro_func fnproc;
    void* arg;
    void* mystack;
    coro_stack stack;
    coro_context coctx;
    coro_context coctx0;
    int state;
    int pkstate;
} goroutine;


void hello(void*arg) {
    int tid = gettid();
    linfo("called %p %d\n", arg, tid);
    goroutine* gr = (goroutine*)arg;
    // linfo("called %p %d %d\n", arg, tid, gr->id);
    // corowp_transfer(&gr->coctx, &gr->coctx0);

    // assert(1==2);
    // sleep(2);
}

static noro* nr;
int main() {
    nr = noro_new();
    noro_init(nr);
    noro_wait_init_done(nr);
    linfo("noro init done %d, %d\n", 12345, gettid());
    sleep(1);
    noro_post(hello, (void*)(uintptr_t)5);
    socket(PF_INET, SOCK_STREAM, 0);
    sleep(5);
    return 0;
}
