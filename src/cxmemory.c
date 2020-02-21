
#include "cxrtbase.h"

extern void* crn_gc_malloc(size_t size);
extern void* crn_gc_realloc(void* ptr, size_t size);
extern void crn_gc_free(void* ptr);
extern void crn_gc_free2(void* ptr);

void* cxmalloc(size_t size) {
    void* ptr = crn_gc_malloc(size);
    return ptr;
}
void* cxrealloc(void*ptr, size_t size) {
    return crn_gc_realloc(ptr, size);
}
void cxfree(void* ptr) {
    crn_gc_free(ptr);
}
void* cxcalloc(size_t blocks, size_t size) {
    return crn_gc_malloc(blocks*size);
}

char* cxstrdup(char* str) {
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

