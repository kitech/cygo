#ifndef _CXRT_BASE_H_
#define _CXRT_BASE_H_


#include <stdio.h>
#include <stdlib.h>
#include <assert.h>
#include <string.h>
#include <pthread.h>
#include <unistd.h>
#include <stdint.h>
#include <stdbool.h>
#include <stdalign.h>
#include <errno.h>
#include <sys/socket.h>

#include <pthread.h>
#include <gc.h> // must put after <pthread.h>

// golang type map
// typedef uint8_t bool;
typedef uint8_t byte;
typedef uint8_t uint8;
typedef uint8_t uchar;
typedef int8_t int8;
typedef uint16_t uint16;
typedef int16_t int16;
typedef uint32_t uint32;
typedef uint32_t rune;
typedef int32_t int32;
typedef uint64_t uint64;
typedef int64_t int64;
typedef float float32;
typedef double float64;
typedef unsigned int uint;
typedef float f32;
typedef double f64;
typedef uint32_t u32;
typedef int32_t i32;
typedef uint16_t u16;
typedef int16_t i16;
typedef uint64_t u64;
typedef int64_t i64;
typedef uintptr_t usize;
typedef uintptr_t uintptr;
// typedef void* error;
typedef void* voidptr;
typedef char* byteptr;
typedef char* charptr; // with tailing 0
typedef void voidty;

#define nilptr NULL
#define iota 0

typedef uint8 metaflag;
typedef int typealg;
typedef struct _metatype {
    usize size;
    voidptr ptrdata;
    uint32 hash;
    metaflag tflag;
    uint8 align;
    uint8 fieldalign;
    uint8 kind;
    typealg alg;
    byteptr gcdata;
    charptr tystr;
    voidptr ptr2this;
} _metatype;
typedef struct cxeface {
    _metatype* _type; // _type
    voidptr data;
} cxeface;
typedef struct ifacetab {
    voidptr inner; // interfacetype*
    _metatype* _type;
    struct ifacetab* link;
    int32 bad;
    int32 inhash;
    usize fun[1];
} ifacetab;
typedef struct cxiface {
    ifacetab* itab; // itab
    voidptr data;
} cxiface;
cxeface cxeface_new_of2(void* data, int sz);
cxeface* cxrt_type2eface(voidptr _type, voidptr data);

// utils
// void println(const char* fmt, ...);
void println2(const char* filename, int lineno, const char* funcname, const char* fmt, ...);
void println3(const char* origfilename, int origlineno, const char* filename, int lineno,
              const char* funcname, const char* fmt, ...);
#define unsafe__Sizeof(x) sizeof(x) // TODO depcreat
#define unsafe__Alignof(x) alignof(x) // TODO depcreat
// #define unsafe__Offsetof(x) offsetof(int, x) // TODO depcreat

// TODO
#define gogorun

extern void cxrt_init_env(int argc, char** argv);
extern void cxrt_fiber_post(void (*fn)(void*), void*arg);
extern void* cxrt_chan_new(int sz);
extern void cxrt_chan_send(void*ch, void*arg);
extern void* cxrt_chan_recv(void*ch);
extern void cxrt_set_finalizer(void*ptr, void(*fn)(void*));

#include <sys/types.h>
extern pid_t cxgettid();

// cxmemory
void* cxmalloc(size_t size);
void* cxrealloc(void*ptr, size_t size);
void cxfree(void* ptr);
void* cxcalloc(size_t nmemb, size_t size);
char* cxstrdup(const char* str);
char* cxstrndup(char* str, int n);
void* cxmemdup(void* ptr, int sz);

#include <collectc/hashtable.h>
#include <collectc/array.h>

// cxstring begin
typedef struct cxstring { char* ptr; int len; } cxstring;
typedef struct cxarray2_s {
    uint8* ptr;  int len;  int cap;  int elemsz; voidptr typ;
} cxarray2;
typedef struct wideptr { voidptr ptr; voidptr obj; } wideptr;

// typedef struct cxstring string;
cxstring* cxstring_new();
cxstring* cxstring_new_cstr(char* s);
cxstring* cxstring_new_cstr2(char* s, int len);
cxstring* cxstring_new_char(char ch);
cxstring* cxstring_add(cxstring* s0, cxstring* s1);
int cxstring_len(cxstring* s);
cxstring* cxstring_sub(cxstring* s0, int start, int end);
bool cxstring_eq(cxstring* s0, cxstring* s1);
bool cxstring_ne(cxstring* s0, cxstring* s1);
char* CString(cxstring* s);
cxstring* GoString(char* s);
cxstring* GoStringN(char* s, int n);
cxstring* cxstring_dup(cxstring* s);
int cxstring_cmp(cxstring* s0, cxstring* s1);
void panic(cxeface* v);
void panicln(cxstring*s);
extern cxstring* cxstring_replace(cxstring* s0, cxstring* old, cxstring* new, int count);

extern cxarray2* cxstring_split(cxstring* s0, cxstring* s1);
extern cxarray2* cxstring_splitch(cxstring* s0, int s1);
// cxstring end

typedef struct error error;
struct error {
    void* thisptr; // error's this object
    cxstring*(*Error)(error*);
};
error* error_new_zero();
cxstring* error_Error(error* err);

// cxhashtable begin
HashTable* cxhashtable_new();
size_t cxhashtable_hash_str(const char *key);
size_t cxhashtable_hash_str2(const char *key, int len);
int HashTable_len(HashTable* ht);
int HashTable_cap(HashTable* ht);
// cxhashtable end

// cxarray begin
Array* cxarray_new();
Array* cxarray_new2(int cap);
Array* cxarray_slice(Array* a0, int start, int end);
void* cxarray_get_at(Array* a0, int idx);
Array* cxarray_append(Array* a0, void* v);

cxarray2* cxarray2_new(int len, int elemsz);
cxarray2* cxarray2_slice(cxarray2* a0, int start, int end);
cxarray2* cxarray2_append(cxarray2* a0, void* v);
voidptr* cxarray2_get_at(cxarray2* a0, int idx);
uint8* cxarray2_replace_at(cxarray2* a0, void* v, int idx, void*out);
void cxarray2_appendn(cxarray2* a0, void* v, int n);

int cxarray2_size(cxarray2* a0);
int cxarray2_len(cxarray2* a0);
int cxarray2_capacity(cxarray2* a0);
int cxarray2_cap(cxarray2* a0);
int cxarray2_elemsz(cxarray2* a0);
#endif

