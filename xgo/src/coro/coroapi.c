
#include "coro.h"

/*
void libcoro_create(void* ctx, void* corofp, void* arg, void* sptr, size_t ssze) {
    coro_create(ctx, corofp, arg, sptr, ssze);
}

void libcoro_transfer(void* prev, void* next) {
    coro_transfer(prev, next);
}
*/

void libcoro_destroy (void *ctx) {
    coro_destroy(ctx);
}

