
// #include <cstdarg>
#include <stdarg.h>
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
// safe when null
int cstrlen(const char* str) { return str==0?0 : strlen(str); }
char* cstrcpy(char* dst, const char* src) {
    if(dst==0||src==0) {return dst; }
    return strcpy(dst, src);
}
char* cstrdup(char* str) {
    int len = cstrlen(str);
    char* ds = cxmalloc(len+1);
    cstrcpy(ds, str);
    return ds;
}

char* cstrndup(char* str, int n) {
    char* ds = cxmalloc(n+1);
    strncpy(ds, str, n);
    return ds;
}

char* cstr_replace_inplace(char* s1, const char*s2, int cnt) {
    return s1;
}
char* cstr_replace_tobuf(const char* s1, const char*s2, char* buf, int len, int cnt) {
    return buf;
}

// Source - https://stackoverflow.com/a
// Posted by jmucchiello, modified by community. See post 'Timeline' for change history
// Retrieved 2025-11-24, License - CC BY-SA 4.0

// You must free the result if result is non-NULL.
char *cstr_replace_newstr(char *orig, char *rep, char *with) {
    char *result; // the return string
    char *ins;    // the next insert point
    char *tmp;    // varies
    int len_rep;  // length of rep (the string to remove)
    int len_with; // length of with (the string to replace rep with)
    int len_front; // distance between rep and end of last rep
    int count;    // number of replacements

    // sanity checks and initialization
    if (!orig || !rep)
        return NULL;
    len_rep = strlen(rep);
    if (len_rep == 0)
        return NULL; // empty rep causes infinite loop during count
    if (!with)
        with = "";
    len_with = strlen(with);

    // count the number of replacements needed
    ins = orig;
    for (count = 0; (tmp = strstr(ins, rep)); ++count) {
        ins = tmp + len_rep;
    }

    tmp = result = cxmalloc(strlen(orig) + (len_with - len_rep) * count + 1);

    if (!result)
        return NULL;

    // first time through the loop, all the variable are set correctly
    // from here on,
    //    tmp points to the end of the result string
    //    ins points to the next occurrence of rep in orig
    //    orig points to the remainder of orig after "end of rep"
    while (count--) {
        ins = strstr(orig, rep);
        len_front = ins - orig;
        tmp = strncpy(tmp, orig, len_front) + len_front;
        tmp = strcpy(tmp, with) + len_with;
        orig += len_front + len_rep; // move to next "end of rep"
    }
    strcpy(tmp, orig);
    return result;
}

char* cstr_replace(char* s1, char*s2, char* s3) {
    char*res = cstr_replace_newstr(s1, s2, s3);
    // printf("%s\n", res);
    return res;
}

char* cstr_substr_inplace(char* s1, int start, int end) {
    return 0;
}
char* cstr_substr_tobuf(char* s1, int start, int end, char* buf, int len) {
    return buf;
}
char* cstr_substr_newstr(char* s1, int start, int end) {
    return 0;
}
char* cstr_substr(char* s1, int start, int end) {
    return cstr_substr_newstr(s1, start, end);
}

char* cstr_trim_left(char* s1, char* s2) {
    return 0;
}
char* cstr_trim_right(char* s1, char* s2) {
    return 0;
}
char* cstr_trim(char* s1, char* s2) {
    return 0;
}
char* cstr_trim_space(char* s1) {
    return 0;
}
char* cstr_trim_left_ch(char* s1, char c) {
    return 0;
}
char* cstr_trim_right_ch(char* s1, char c) {
    return 0;
}
char* cstr_trim_ch(char* s1, char c) {
    return 0;
}

char** cstr_split_newstr(char* s1, const char* s2) {
    return 0;
}
char** cstr_split(char* s1, const char* s2) {
    return 0;
}

// api is a macro cstrcat(s1, ...)
char* cstrcat_impl(char* s1, int count, ...) {
    char* s = cstrdup(s1);
    va_list args;
    va_start(args, count);

    int len = cstrlen(s);
    for (int i=0; i<count; i++) {
        char* s2 = va_arg(args, char*);
        char* t = cxrealloc(s, len+1 + cstrlen(s2));
        cstrcpy(t+len, s2);
        s = t;
        len += cstrlen(s2);
    }
    va_end(args);
    return s;
}
char* cstrcat0(char* s1, const char* s2) {
    int size = cstrlen(s1)+cstrlen(s2)+1;
    char* ptr = cxmalloc(size);
    cstrcpy(ptr, s1);
    cstrcpy(ptr+cstrlen(s1), s2);
    return ptr;
}

// api is a macro cstrjoin(sep, ...)
char* cstrjoin_impl(char* sep, int count, ...) {
    char *s = 0;
    va_list args;
    va_start(args, count);

    for(int i=0; i<count; i++) {
        char* s2 = va_arg(args, char*);
        s = cstrcat(s, s2);
        if(i<count-1) s = cstrcat_impl(s, 1, sep);
    }
    va_end(args);
    return s;
}

void* cxmemdup(void* ptr, int sz) {
    void* dp = cxmalloc(sz);
    memcpy(dp, ptr, sz);
    return dp;
}
