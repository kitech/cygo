#include <assert.h>
#include <stdarg.h>
#include <pthread.h>
#include <string.h>
#include <stdio.h>
// #include <ucontext.h>

// !!!!!
// on macos, coro.h conflict with stdlib.h
// cause sizeof(ucontext_t) much small(56) than correct value(768)
#ifdef __APPLE__
// so fork
#else
#include <stdlib.h>
#endif
#include "coro.h"

#include "corona_util.h"
#include "coronapriv.h"
#include "futex.h"

// core functions 有些是宏，所以就再包一下
// 对于non threadsafe的函数，做了简单lock

extern void* crn_gc_malloc(size_t size);

#ifdef __APPLE__
// #define _XOPEN_SOURCE
// #include <ucontext.h>
// int swapcontext(ucontext_t *, const ucontext_t *);
#endif
// #include <ucontext.h>

coro_context* corowp_context_new() {
    coro_context* ctx = (coro_context*)crn_gc_malloc(sizeof(coro_context));
    return ctx;
}

__thread coro_context *crn_main_coro_ctx = 0;
void corowp_set_main_ctx(coro_context* ctx) {
    crn_main_coro_ctx = ctx;
}

// #define CORO_BACKEND_LIBCO
#define CORO_BACKEND_UCONEXT
// https://github.com/semistrict/libcoro
// #define CORO_BACKEND_LIBCORO

// 加锁逻辑是错的，这个函数就像开始调用一个函数一样，可以多线程并发调用的。
// 如果真的需要同步调用，那么也还是要考虑在上层视逻辑需要决定是否加锁。
static pmutex_t coroccmu = PTHREAD_MUTEX_INITIALIZER;

#if defined (CORO_BACKEND_LIBCO)

#include "libco/libco.h"

typedef struct libco_context {
    coro_func usrthr;
    void* usrarg;
    int hasarg;
    cothread_t cothr;
}libco_context;

static __thread void* libco_arg = 0;
void co_switch_with_arg(cothread_t thread, void* arg) {
    libco_arg = arg;
    co_switch(thread);
}

void libco_entry_point() {
    libco_context* ctx = libco_arg;
    libco_arg = 0;
    coro_func usrthr = ctx->usrthr;
    void* usrarg = ctx->usrarg;
    linfo("run usrthr %p, usrarg %p, thr %p\n", ctx->usrthr, ctx->usrarg, ctx->cothr);

    usrthr(usrarg);
}

void corowp_create(coro_context *ctx, coro_func coro, void *arg, void *sptr,  size_t ssze) {
    assert(sizeof(coro_context)>=sizeof(libco_context));
    printf("corowp_create %p %p %p %p %lu\n", ctx, coro, arg, sptr, ssze);
    assert(ctx != nilptr);
    if (coro==0) assert(arg == nilptr && sptr==0 && ssze==0 );
    pmutex_lock(&coroccmu);
    libco_context rctx = {0};
    rctx.hasarg = 42;
    rctx.usrthr = coro;
    rctx.usrarg = arg;
    rctx.cothr = co_create(ssze, libco_entry_point);
    linfo("create coro with fn %p, arg %p thr %p\n", rctx.usrthr, rctx.usrarg, rctx.cothr);
    memcpy(ctx, &rctx, sizeof(libco_context));
    pmutex_unlock(&coroccmu);
}

void corowp_transfer(coro_context *prev, coro_context *next) {
    libco_context* rctx0 = (libco_context*)prev;
    libco_context* rctx1 = (libco_context*)next;
    if (rctx0->cothr==0) {
        // mainco => workco
        if (rctx1->hasarg == 42) {
            linfo("first pass arg %d, fn %p, arg %p, thr %p\n", rctx1->hasarg, rctx1->usrthr, rctx1->usrarg, rctx1->cothr);
            rctx1->hasarg = 43;
            co_switch_with_arg(rctx1->cothr, rctx1);
        }else{
             co_switch_with_arg(rctx1->cothr, rctx1);
            // co_switch(rctx1->cothr);
        }
    } else if (rctx1->cothr==0) {
        // workco => mainco
        co_switch(co_active());
    } else {
        assert(0);
    }
}

void corowp_destroy (coro_context *ctx) {
    // coro_destroy(ctx);
}

#elif defined (CORO_BACKEND_UCONEXT)

// makecontext need this proto type
static void corowp_ucontext_corofwd(int arg0, ...) {
    int argc = arg0;
    assert(argc==2); // arg0=2, arg1=fn, arg2=arg
    void* (*f)(void*) = 0;
    void* arg = 0;

    va_list list;
    va_start(list, arg0);
    f = va_arg(list, void*);
    arg = va_arg(list, void*);
    va_end(list);

    printf("arg0 %d, f %p, arg %p\n", arg0, f, arg);
    assert(f!=0);
    f(arg);
}

void corowp_create(coro_context *ctx, coro_func coro, void *arg, void *sptr,  size_t ssze) {
    // ucontext_t should 700+, why so small???
    assert(sizeof(coro_context)>=sizeof(ucontext_t));
    printf("sizeof(coro_context)=%lu, sizeof(ucontext_t)=%lu\n", sizeof(coro_context), sizeof(ucontext_t));
    assert(sizeof(ucontext_t)>700); // linux 900+, mac 700+

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

    makecontext(rctx, (void*) corowp_ucontext_corofwd, 3, 2, (void*)coro, arg);
    // makecontext(rctx, (void*) coro, 1, arg);
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
