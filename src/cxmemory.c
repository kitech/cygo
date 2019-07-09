
#include "cxrtbase.h"

void* cxmalloc(size_t size) {
    void* ptr = GC_MALLOC(size);
    return ptr;
}
void* cxrealloc(void*ptr, size_t size) {
    return GC_REALLOC(ptr, size);
}
void cxfree(void* ptr) {
    GC_FREE(ptr);
}
void* cxcalloc(size_t blocks, size_t size) {
    return cxmalloc(blocks*size);
}

char* cxstrdup(const char* str) {
    char* ds = cxmalloc(strlen(str)+1);
    strcpy(ds, str);
    return ds;
}

char* cxstrndup(char* str, int n) {
    char* ds = cxmalloc(n+1);
    strncpy(ds, str, n);
    return ds;
}

void* cxmemdup(void* ptr, int sz) {
    void* dp = cxmalloc(sz);
    memcpy(dp, ptr, sz);
    return dp;
}

