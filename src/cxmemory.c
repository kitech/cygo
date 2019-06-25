
#include "cxpriv.h"

void* cxmalloc(int size) {
    return calloc(1, size);
}
void* cxrealloc(void*ptr, int size) {
    return realloc(ptr, size);
}
void cxfree(void* ptr) {
    free(ptr);
}

