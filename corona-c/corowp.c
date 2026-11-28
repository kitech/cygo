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

// ---- coro backend selection: Android uses vendored libco, others ucontext ----
// Android: vendored higan-emu/libco
//   repo:   https://github.com/higan-emu/libco
//   commit: e18e09d634d612a01781168ad4d76be10a7e3bad
//   date:   2024-09-09 (libco v20, ISC)
// Linux/macOS: POSIX ucontext
#if defined(CORO_LIBCO89)
  #include "../3rdparty/libco/libco.h"
#else
  #define CORO_BACKEND_UCONEXT
#endif

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
#if defined(CORO_LIBCO89)
    co_active(); // initialise libco's thread-local primary context
#endif
}

// backend is selected above: CORO_LIBCO -> libco, otherwise ucontext

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

#elif defined (CORO_LIBCO89)

static __thread coro_context* crn_libco_boot_ctx;
// one-shot handoff: corowp_create writes, the entry consumes. create is
// always immediately followed by transfer on the same machine thread.
static __thread int crn_libco_boot_used;

static void corowp_libco_entry(void) {
    assert(crn_libco_boot_used == 1);
    coro_context* ctx = crn_libco_boot_ctx;
    crn_libco_boot_used = 0; // consume before running user code
    assert(ctx != nilptr);
    assert(ctx->coro != 0);
    ctx->coro(ctx->arg);
    // corowp_transfer should have switched away; falling out of the entry
    // would return into garbage. keep it visible instead of assert(NDEBUG-noop)
    fiber* gr = crn_fiber_getcur();
    lerror("fiber entry returned (libco): grid=%d mcid=%d\n",
           gr ? gr->id : -1, gr ? gr->mcid : -1);
    abort();
}

void corowp_create(coro_context *ctx, coro_func coro, void *arg, void *sptr, size_t ssze) {
    assert(ctx != nilptr);
    ctx->coro = coro;
    ctx->arg = arg;
    /* Layout (low -> high): [handle page][guard page][stack ...].
     * libco writes its handle block at `sptr`; the guard above it faults
     * before a stack overflow can corrupt the saved context. */
    assert(crn_libco_boot_used == 0);
    crn_libco_boot_ctx = ctx; // thread-local: create is followed by transfer
    crn_libco_boot_used = 1;
    ctx->handle = co_derive(sptr, (unsigned int)ssze, corowp_libco_entry);
    assert(ctx->handle != NULL);
}

void corowp_transfer(coro_context *prev, coro_context *next) {
    prev->handle = co_active(); // libco saves current state into co_active_handle
    co_switch(next->handle);
}

void corowp_destroy (coro_context *ctx) {
    // stack is released by corona's munmap, not co_delete()
}

#elif defined (CORO_BACKEND_UCONEXT)

// makecontext's int varargs cannot portably carry pointers (LP64 truncation
// per POSIX), so the entry reads its {fn,arg} from a one-shot thread-local
// slot written by corowp_create. Invariant: create is always immediately
// followed by transfer on the same machine thread (crn_fiber_run_first), so
// the slot cannot be overwritten before it is consumed; the used flag turns
// a future violation of that invariant into an assert.
typedef struct {
    coro_func fn;
    void* arg;
    int used;
} crn_uc_boot_t;
static __thread crn_uc_boot_t crn_uc_boot;

static void corowp_ucontext_corofwd(void) {
    assert(crn_uc_boot.used == 1);
    coro_func f = crn_uc_boot.fn;
    void* a = crn_uc_boot.arg;
    crn_uc_boot.used = 0; // consume before running user code
    crn_uc_boot.fn = 0;

    ldebug("f %p, arg %p\n", (void*)f, a);
    assert(f!=0);
    f(a);
    // corowp_transfer should have switched away; falling out of the entry
    // with uc_link==NULL would silently exit(0) the whole process, and with
    // uc_link set it would masquerade as a normally-finished fiber. both
    // are state-machine violations: make them loud.
    fiber* gr = crn_fiber_getcur();
    lerror("fiber entry returned: grid=%d mcid=%d\n",
           gr ? gr->id : -1, gr ? gr->mcid : -1);
    abort();
}

void corowp_create(coro_context *ctx, coro_func coro, void *arg, void *sptr,  size_t ssze) {
    // ucontext_t should 700+, why so small???
    assert(sizeof(coro_context)>=sizeof(ucontext_t));
    ldebug("sizeof(coro_context)=%lu, sizeof(ucontext_t)=%lu\n", sizeof(coro_context), sizeof(ucontext_t));
    if (sizeof(void*) <= 4) {
	assert(sizeof(ucontext_t)>300); // linux 900+, mac 700+
    } else {
	assert(sizeof(ucontext_t)>700); // linux 900+, mac 700+
    }

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
    /* Backstop only: the normal end-of-fiber path is the explicit
     * corowp_transfer in crn_fiber_forward. If the entry ever returned
     * anyway, uc_link must not be NULL: glibc terminates the whole process
     * with exit(0) for a NULL uc_link, i.e. a silent, "successful" exit.
     * (The entry itself aborts before falling through to this.) */
    rctx->uc_link = (ucontext_t*)crn_main_coro_ctx;
    assert (crn_main_coro_ctx != 0);

    assert(crn_uc_boot.used == 0);
    crn_uc_boot.fn = coro;
    crn_uc_boot.arg = arg;
    crn_uc_boot.used = 1;
    makecontext(rctx, (void*) corowp_ucontext_corofwd, 1, 0);
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
