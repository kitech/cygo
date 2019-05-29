#include "norogc.h"

void* noro_malloc(size_t size) {
    return NORO_MALLOC(size);
}
void noro_free(void* ptr){
    NORO_FREE(ptr);
}
void* noro_realloc(void* obj, size_t new_size){
    return NORO_REALLOC(obj, new_size);
}

