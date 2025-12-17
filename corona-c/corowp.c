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

// #define CORO_BACKEND_FCONTEXT
#define CORO_BACKEND_UCONEXT
// https://github.com/semistrict/libcoro
// #define CORO_BACKEND_LIBCORO

// 加锁逻辑是错的，这个函数就像开始调用一个函数一样，可以多线程并发调用的。
// 如果真的需要同步调用，那么也还是要考虑在上层视逻辑需要决定是否加锁。
static pmutex_t coroccmu = PTHREAD_MUTEX_INITIALIZER;

#if defined (CORO_BACKEND_FCONTEX)

typedef void* fcontext_t;
typedef struct ftransfer_t {
    fcontext_t fctx;
    void* data;
} ftransfer_t;
typedef struct ftransfer_data_t {
    ftransfer_t ftran;
    coro_func usrthr;
    void* usrarg;
    int hasarg;
} ftransfer_data_t;

fcontext_t (make_fcontext_f)(void* sp, size_t size, void(*f)(ftransfer_t)) = 0;
ftransfer_t (jump_fcontext_f)(fcontext_t fctx, void* data) = 0;

void fcontext_entry_point(ftransfer_t ftran) {
    ftransfer_data_t* ctx = ftran.data;
    coro_func usrthr = ctx->usrthr;
    void* usrarg = ctx->usrarg;
    linfo("run usrthr %p, usrarg %p, thr %p\n", ctx->usrthr, ctx->usrarg, ctx->cothr);

    usrthr(usrarg);
}

void corowp_create(coro_context *ctx, coro_func coro, void *arg, void *sptr,  size_t ssze) {
    assert(sizeof(coro_context)>=sizeof(ftransfer_data_t));
    printf("corowp_create %p %p %p %p %lu\n", ctx, coro, arg, sptr, ssze);
    assert(ctx != nilptr);
    if (coro==0) assert(arg == nilptr && sptr==0 && ssze==0 );

    fcontext_t fctx = make_fcontext_f(sptr, ssze, fcontext_entry_point);
    ftransfer_data_t *rctx = ctx;
    rctx->ftran.fctx = fctx;
    rctx->ftran.data = ctx; // reverse pointer to parent struct
    rctx->hasarg = 42;
    rctx->usrthr = coro;
    rctx->usrarg = arg;
    rctx->cothr = fctx;
    linfo("created fcontext with fn %p, arg %p thr %p\n", rctx.usrthr, rctx.usrarg, rctx.cothr);

}

void corowp_transfer(coro_context *prev, coro_context *next) {
    ftransfer_data_t* rctx0 = (ftransfer_data_t*)prev;
    ftransfer_data_t* rctx1 = (ftransfer_data_t*)next;
    if (rctx0->cothr==0) {
        // mainco => workco
        linfo("first pass arg %d, fn %p, arg %p, thr %p\n", rctx1->hasarg, rctx1->usrthr, rctx1->usrarg, rctx1->cothr);        ftransfer_t ftran = jump_fcontext_f(rctx1->ftran);
        rctx0->ftran = jump_fcontext_f(rctx1->ftran);
    } else if (rctx1->cothr==0) {
        // workco => mainco
        ftransfer_t ftran = jump_fcontext_f(rctx1->ftran);
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

    ldebug("arg0 %d, f %p, arg %p\n", arg0, f, arg);
    assert(f!=0);
    f(arg);
}

void corowp_create(coro_context *ctx, coro_func coro, void *arg, void *sptr,  size_t ssze) {
    // ucontext_t should 700+, why so small???
    assert(sizeof(coro_context)>=sizeof(ucontext_t));
    ldebug("sizeof(coro_context)=%lu, sizeof(ucontext_t)=%lu\n", sizeof(coro_context), sizeof(ucontext_t));
    assert(sizeof(ucontext_t)>700); // linux 900+, mac 700+

    ldebug("corowp_create %p %p %p %p %lu\n", ctx, coro, arg, sptr, ssze);
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
