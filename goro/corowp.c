#include <stdlib.h>
#include "coro.h"

// core functions 有些是宏，所以就再包一下
// 对于non threadsafe的函数，做了简单lock

coro_context* corowp_context_new() {
    coro_context* ctx = (coro_context*)calloc(1, sizeof(coro_context));
    return ctx;
}

static pthread_mutex_t coroccmu;
void corowp_create(coro_context *ctx, coro_func coro, void *arg, void *sptr,  size_t ssze) {
    pthread_mutex_lock(&coroccmu);
    coro_create(ctx, coro, arg, sptr, ssze);
    pthread_mutex_unlock(&coroccmu);
}

void corowp_transfer(coro_context *prev, coro_context *next) {
    coro_transfer(prev, next);
}

void corowp_destroy (coro_context *ctx) { coro_destroy(ctx); }

#if CORO_STACKALLOC
typedef struct coro_stack coro_stack;

coro_stack* corowp_stack_new() {
    coro_stack* stk = (coro_stack*)calloc(1, sizeof(coro_stack));
    return stk;
}
int corowp_stack_alloc (coro_stack *stack, unsigned int size) {
    return coro_stack_alloc(stack, size);
}
void corowp_stack_free(coro_stack* stack) {
    coro_stack_free (stack);
}
#endif
