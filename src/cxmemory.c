
#include <stdlib.h>
#include <string.h>
#include <dlfcn.h>

#include "cxrtbase.h"
#include "cxtypedefs.h"

// extern void* __real_malloc(size_t);
// extern void* __real_calloc(size_t, size_t);
// extern void* __real_realloc(void*, size_t);
// extern void* __real_aligned_alloc(size_t, size_t);
// extern void __real_free(void*);

Allocator cxaltrc = {.malloc=malloc, .calloc=calloc, .realloc=realloc, .free=free};

// #ifdef USE_BDWGC
extern void* crn_gc_malloc(size_t size);
extern void* crn_gc_calloc(size_t num, size_t size);
extern void* crn_gc_realloc(void* ptr, size_t size);
extern void crn_gc_free(void* ptr);
extern void crn_gc_free2(void* ptr);

Allocator cxaltgc = {.malloc=crn_gc_malloc, .calloc=crn_gc_calloc,
                    .realloc=crn_gc_realloc,.free=crn_gc_free};
// #endif

static Allocator* cxalt_ = 1 ? (&cxaltgc) : (&cxaltrc);

void* cxmalloc(size_t size) {
    void* ptr = cxalt_->malloc(size);
    // void* ptr = crn_gc_malloc(size);
    return ptr;
}
void* cxrealloc(void*ptr, size_t size) {
    return cxalt_->realloc(ptr, size);
    // return crn_gc_realloc(ptr, size);
}
void cxfree(void* ptr) {
    cxalt_->free(ptr);
    // crn_gc_free(ptr);
}
void* cxcalloc(size_t blocks, size_t size) {
    return cxalt_->calloc(blocks, size);
    // return crn_gc_malloc(blocks*size);
}

/////

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

char* cxstr_replace_inplace(char* s1, const char*s2, int cnt) {
    return s1;
}
char* cxstr_replace_tobuf(const char* s1, const char*s2, char* buf, int len, int cnt) {
    return buf;
}
char* cxstr_replace_newstr(const char* s1, const char*s2, int cnt) {
    return 0;
}
char* cxstr_replace(const char* s1, const char*s2, int cnt) {
    return cxstr_replace_newstr(s1, s2, cnt);
}

char* cxstr_substr_inplace(char* s1, int start, int end) {
    return 0;
}
char* cxstr_substr_tobuf(char* s1, int start, int end, char* buf, int len) {
    return buf;
}
char* cxstr_substr_newstr(char* s1, int start, int end) {
    return 0;
}
char* cxstr_substr(char* s1, int start, int end) {
    return cxstr_substr_newstr(s1, start, end);
}

char* cxstr_trim_left(char* s1, char* s2) {
    return 0;
}
char* cxstr_trim_right(char* s1, char* s2) {
    return 0;
}
char* cxstr_trim(char* s1, char* s2) {
    return 0;
}
char* cxstr_trim_space(char* s1) {
    return 0;
}
char* cxstr_trim_left_ch(char* s1, char c) {
    return 0;
}
char* cxstr_trim_right_ch(char* s1, char c) {
    return 0;
}
char* cxstr_trim_ch(char* s1, char c) {
    return 0;
}

char** cxstr_split_newstr(char* s1, const char* s2) {
    return 0;
}
char** cxstr_split(char* s1, const char* s2) {
    return 0;
}

void* cxmemdup(void* ptr, int sz) {
    void* dp = cxmalloc(sz);
    memcpy(dp, ptr, sz);
    return dp;
}
