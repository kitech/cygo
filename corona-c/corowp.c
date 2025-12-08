#include <assert.h>
#include <pthread.h>
#include <stdlib.h>
#include <sys/ucontext.h>

#include "coro.h"
#include "corona_util.h"
#include "coronapriv.h"
#include "futex.h"

// core functions 有些是宏，所以就再包一下
// 对于non threadsafe的函数，做了简单lock

extern void* crn_gc_malloc(size_t size);

#ifdef __APPLE__
#define _XOPEN_SOURCE
#include <ucontext.h>
int swapcontext(ucontext_t *, const ucontext_t *);
#endif
#include <ucontext.h>

coro_context* corowp_context_new() {
    coro_context* ctx = (coro_context*)crn_gc_malloc(sizeof(coro_context));
    return ctx;
}

__thread coro_context *crn_main_coro_ctx = 0;
void corowp_set_main_ctx(coro_context* ctx) {
    crn_main_coro_ctx = ctx;
}

// 加锁逻辑是错的，这个函数就像开始调用一个函数一样，可以多线程并发调用的。
// 如果真的需要同步调用，那么也还是要考虑在上层视逻辑需要决定是否加锁。
static pmutex_t coroccmu = PTHREAD_MUTEX_INITIALIZER;
#if 1
void corowp_create(coro_context *ctx, coro_func coro, void *arg, void *sptr,  size_t ssze) {
    assert(sizeof(coro_context)>=sizeof(ucontext_t));
    printf("corowp_create %p %p %p %p %lu\n", ctx, coro, arg, sptr, ssze);
    assert(ctx != nilptr);
    if (coro==0) assert(arg == nilptr && sptr==0 && ssze==0 );
    pmutex_lock(&coroccmu);
    ucontext_t* rctx = (ucontext_t*)ctx;
    int rv = getcontext(rctx);
    if ( rv == -1) {
        lerror("getcontext error %d\n", rv);
    }

    rctx->uc_stack.ss_sp = sptr;
    rctx->uc_stack.ss_size = ssze;
    rctx->uc_link = 0; // (ucontext_t*)crn_main_coro_ctx; // ???
    assert (crn_main_coro_ctx != 0);

    makecontext(rctx, (void*) coro, 1, arg);
    pmutex_unlock(&coroccmu);
}

void corowp_transfer(coro_context *prev, coro_context *next) {
    // coro_transfer(prev, next);
    int rv = swapcontext((ucontext_t*) prev, (ucontext_t*) next);
    if (rv == -1) {
        lerror("swapcontext error %d\n", rv);
    }
}

void corowp_destroy (coro_context *ctx) {
    // coro_destroy(ctx);
}
#else
void corowp_create(coro_context *ctx, coro_func coro, void *arg, void *sptr,  size_t ssze) {
    printf("corowp_create %p %p %p %p %lu\n", ctx, coro, arg, sptr, ssze);
    assert(ctx != nilptr);
    if (coro==0) assert(arg == nilptr && sptr==0 && ssze==0 );
    pmutex_lock(&coroccmu);
    coro_create(ctx, coro, arg, sptr, ssze);
    pmutex_unlock(&coroccmu);
}

void corowp_transfer(coro_context *prev, coro_context *next) {
    coro_transfer(prev, next);
}

void corowp_destroy (coro_context *ctx) { coro_destroy(ctx); }
#endif

#if CORO_STACKALLOC
typedef struct coro_stack coro_stack;

coro_stack* corowp_stack_new() {
    coro_stack* stk = (coro_stack*)crn_gc_malloc(sizeof(coro_stack));
    return stk;
}
int corowp_stack_alloc (coro_stack *stack, unsigned int size) {
    return coro_stack_alloc(stack, size);
}
void corowp_stack_free(coro_stack* stack) {
    coro_stack_free (stack);
}
#endif
