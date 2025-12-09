#ifndef _CXRT_TYPEDEFS_H_
#define _CXRT_TYPEDEFS_H_

#include <stdlib.h>
#include <stdint.h>
#include <stdbool.h>
#include <stdalign.h>

/// sys/os/compiler/stdlib

// compiler test demo
#ifdef __TINYC__
#endif
#if defined(__GNUC__)
#endif

// arch test demo
#if defined(__x86_64__) || defined(__amd64__) || defined(_M_X64)
    // Code specific to x86-64 architecture
    #define CX_ARCH_X64
#elif defined(__i386__) || defined(_M_IX86)
    // Code specific to 32-bit x86 architecture
    #define CX_ARCH_X86
#elif defined(__aarch64__) || defined(_M_ARM64)
    // Code specific to ARM64 architecture
    #define CX_ARCH_ARM64
#elif defined(__arm__)
    // Code specific to 32-bit ARM architecture
    #define CX_ARCH_ARM
#else
    #error "Unsupported architecture"
#endif

#define	__hidden	__attribute__((__visibility__("hidden")))
#define	__exported	__attribute__((__visibility__("default")))
#define	__noinline	__attribute__ ((__noinline__))
#ifndef __always_inline
#define	__always_inline	__attribute__((__always_inline__))
#endif

#if defined(__clang__) || defined (__GNUC__)
    #define HAVE_AUTO_TYPE
    #define HAVE_THREAD_LOCAL
    #define HAVE___THREAD
    // #define HAVE_VA_OPT
#else
#endif

// cxthread_local, cxtls_def, cxtls_set, cxtls_get
// check _Alignas, _Alignof, _Atomic, _Thread_local
#define cxatomic _Atomic
#define cxalignas _Alignas
#define cxalignof _Alignof

#ifdef HAVE_THREAD_LOCAL
    #define cxthread_local _Thread_local
#elif defined(HAVE___THREAD)
    #define cxthread_local __thread
#else
    #warning "not support _Thread_local, use pthread tls instead, but not portable"
// #define	_Thread_local _Atomic
#endif
#ifdef cxthread_local
    #define cxtls_def(varty, var) cxthread_local varty var;
    #define cxtls_get(var) (var)
    #define cxtls_set(var, value) var = value
#else
    // noreturn, global scope
    #define cxtls_def(varty, var) static usize var = 0; \
            static int var##_inited = 0; \
            static cxatomic varty var##_typed;
    // return varty
    #define cxtls_get(var) ({ \
        if (!var##_inited) { var##_inited=1; int rc = pthread_key_create(&(var), NULL); assert(rc==0); } \
        void* pv = pthread_getspecific((var)); \
        __typeof__(var##_typed) tv; memcpy((void*)&tv, &pv, sizeof(tv)); \
        tv; \
    })
    // return int status
    #define cxtls_set(var, value) ({ \
        if (sizeof(var##_typed)>sizeof(void*)) { cxpanic(-1, "value large than sizeof(void*)"); }  \
        if (!var##_inited) { var##_inited=1; int rc = pthread_key_create(&(var), NULL); assert(rc==0); } \
        __typeof__(var##_typed) tv = value; void* voidval=NULL; memcpy(&voidval, (void*)&tv, sizeof(tv)); \
        int rc = pthread_setspecific((var), voidval); \
        rc; \
    })

#endif

// todo cxauto
#ifndef HAVE_AUTO_TYPE
// #error "__auto_type not support"
#warning "not support __auto_type"
// not support func var
#define autodef(var, right_expr) __typeof__((right_expr)) var = (right_expr)
#else
#define autodef(var, right_expr) __auto_type var = right_expr
#define cxauto __auto_type
#endif

////// utils

#define cxmin(a0, a1) ((a0) < (a1)) ? (a0) : (a1))
#define cxmax(a0, a1) ((a0) > (a1)) ? (a0) : (a1))
#define cxswap(a, b) ({__typeof__(a) tmp_ = a; a=b; b=tmp_; })
// readonly cxstr_xxx, for need malloc see cxmemory.c
#define cstreq(s1, s2) strcmp((s1), (s2))==0
#define cstrne(s1, s2) strcmp((s1), (s2))!=0
#define cstr_haspfx(s1, s2) (strlen(s1)>=strlen(s2) && 0 == memcmp((s1), (s2), strlen(s2)))
#define cstr_hassfx(s1, s2) (strlen(s1)>=strlen(s2) && 0 == memcmp(s1+(strlen(s1)-strlen(s2)), (s2), strlen(s2)))
#define cstr_hasstr(s1, s2) strstr(s1, s2) != NULL
#define cstr_index(s1, s2) ({char* tmp_ = strstr(s1, s2); (tmp_==0 ? -1: tmp_ - s1); })
#define cstr_index_ch(s1, ch) ({char* tmp_ = strchr(s1, ch); (tmp_==0 ? -1: tmp_ - s1); })
#define cstr_rindex(s1, s2) ({char* tmp_ = strrstr(s1, s2); (tmp_==0 ? -1: tmp_ - s1); })
#define cstr_rindex_ch(s1, ch) ({char* tmp_ = strrchr(s1, ch); (tmp_==0 ? -1: tmp_ - s1); })
#define cstr_count(s1, s2) ({int cnt_=0; for(char* tmp_=strstr(s1,s2); tmp_!=0; tmp_=strstr(tmp_,s2)) {cnt_++;} cnt_; })
#define cstr_tolower(s) for(char*p=s;*p!=0;p++){if(*p>='A'&&*p<='Z') *p += ('z'-'a'); }
#define cstrs_hasstr(arr, s2) ({int has_=0; for(char*tmp_=arr; tmp_!=0; tmp_++) {if(cxstreq(tmp_,s2)) { has_=1; break;} } has_; })
#define cstrs_index(arr, s2) ({int idx_=-1; int cnt_; for(char*tmp_=arr; tmp_!=0; tmp_++, cnt_++) {if(cxstreq(tmp_,s2)) { idx_=cnt_; break;} } idx_; })
#define cstrs_len(arr) ({int cnt_; for(char*tmp_=arr; tmp_!=0; tmp_++,cnt_++){} cnt_;})
extern void cstrs_sort(char** arr, int desc);
// #define cxstrs_reverse(arr) ({})


/////// extend

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
// typedef intptr_t isize;
typedef long isize;
typedef long long longlong ;
typedef unsigned long ulong;
typedef unsigned long long ulonglong ;
// typedef void* error;
typedef void* voidptr;
typedef void* voidstar;
typedef uint8* byteptr;
typedef char* charptr; // with tailing 0
typedef char** charptrptr;
typedef void voidty;
typedef void unit;

#ifndef nilptr
#define nilptr NULL
#endif
#define cnull NULL
#define null NULL
#define nil NULL
#define iota 0

// IDTSTR(int) => "int"
#define IDTSTR(idt) ""#idt
#define IDTLEN(idt) sizeof(""#idt)
#define IDTCONCAT(idt1, idt2) idt1##idt2
// #define ESCHASH(hashch, ...) hashch __VA_ARGS__
// #define COMPTIME_ERROR(msg) ESCHASH(#, error msg) // not works

#define ctypeid_all 1
enum ctypeid {
    ctypeid_none = iota      ,
    ctypeid_other = iota + 10,
    ctypeid_bool,
    ctypeid_char,
    ctypeid_uchar,
    ctypeid_short,
    ctypeid_ushort,
    ctypeid_int,
    ctypeid_uint,
    ctypeid_long,
    ctypeid_ulong,
    ctypeid_longlong,
    ctypeid_ulonglong,
    ctypeid_float,
    ctypeid_double,
    ctypeid_longdouble,
    ctypeid_charptr,
    ctypeid_charptrptr,
    ctypeid_voidptr,
    ctypeid_voidptrptr,
    ctypeid_intptr,
    ctypeid_func_void, // void,(*)()
    ctypeid_func_int, // int(*,)()
    ctypeid_func_int8, // int(*,)()
    ctypeid_func_int32, // int(*,)()
    ctypeid_func_int64, // int(*,)()
    ctypeid_func_float, // int(*,)()
    ctypeid_func_double, // int(*,)()
    ctypeid_func_charptr, // int(*,)()
    ctypeid_func_voidptr, // int(*,)()
    ctypeid_func_usize, // int(*,)()
    ctypeid_func_isize, // int(*,)()
    ctypeid_user = 65535,
};

// extend +++

typedef struct mintype {
    int ctypeid;
    int size;
    char* fmtstr;
    int fmtstrid; // which one is better
    char* (*tostr)(void* valp); // convert to string func
} mintype;

#include "../3rdparty/va_args_iterators/pp_iter.h"
#include "../3rdparty/simpinc/VA_OPT.h"
#include "cxmacro_p.h"
#define ctypeidof(x) ctypeidof_priv(x)
#define cxpanic(code, msg) cxpanic_priv(code, msg)
#define cxpanicif(cond, code, msg) if(cond) { cxpanic_priv(code, msg); }
#define cxunreach() cxunreach_priv()
extern const char* ctypeid_tostr(int);
extern const char* ctypeid_tofmt(int);


typedef struct Allocator_s {
    void* (*malloc)(usize n);
    void* (*calloc)(usize num, usize size);
    void* (*realloc)(void* ptr, usize n);
    void (*free)(void* ptr);
} Allocator;
// extern Allocator cxaltrc;
// extern Allocator cxaltgc;

#endif // _CXRT_TYPEDEFS_H_
