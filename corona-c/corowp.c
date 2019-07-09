#include <stdlib.h>

#include "coro.h"
#include "futex.h"

// core functions 有些是宏，所以就再包一下
// 对于non threadsafe的函数，做了简单lock

extern void* crn_gc_malloc(size_t size);

coro_context* corowp_context_new() {
    coro_context* ctx = (coro_context*)crn_gc_malloc(sizeof(coro_context));
    return ctx;
}

// 加锁逻辑是错的，这个函数就像开始调用一个函数一样，可以多线程并发调用的。
// 如果真的需要同步调用，那么也还是要考虑在上层视逻辑需要决定是否加锁。
static pmutex_t coroccmu;
void corowp_create(coro_context *ctx, coro_func coro, void *arg, void *sptr,  size_t ssze) {
    pmutex_lock(&coroccmu);
    coro_create(ctx, coro, arg, sptr, ssze);
    pmutex_unlock(&coroccmu);
}

void corowp_transfer(coro_context *prev, coro_context *next) {
    coro_transfer(prev, next);
}

void corowp_destroy (coro_context *ctx) { coro_destroy(ctx); }

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
