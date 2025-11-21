#ifndef _CXMACRO_P_H_
#define _CXMACRO_P_H_

#include <stdio.h>

// literal 'x' matched to int, just ((char)'x') works
// only for basic c type
// not supported
// char arr[], error: unknown type size
#define ctypeidof_priv(x) \
    _Generic((x),        /* Get the name of a type */             \
            _Bool: ctypeid_bool,                  unsigned char: ctypeid_uchar,          \
             char: ctypeid_char,                    signed char: ctypeid_char,            \
        short int: ctypeid_short,            unsigned short int: ctypeid_ushort,     \
              int: ctypeid_int,                    unsigned int: ctypeid_uint,           \
         long int: ctypeid_long,              unsigned long int: ctypeid_ulong,      \
    long long int: ctypeid_longlong,     unsigned long long int: ctypeid_ulonglong, \
            float: ctypeid_float,                        double: ctypeid_double,         \
      long double: ctypeid_longdouble,                   char *: ctypeid_charptr,        \
           void *: ctypeid_voidptr,                       int *: ctypeid_intptr,         \
          char **: ctypeid_charptrptr,                  void **: ctypeid_voidptrptr,     \
        int(*)(): ctypeid_func_int,                   void(*)(): ctypeid_func_void,          \
        char*(*)(): ctypeid_func_charptr,            void*(*)(): ctypeid_func_voidptr,          \
        float(*)(): ctypeid_func_float,             double(*)(): ctypeid_func_double,          \
        uint64(*)(): ctypeid_func_int64,              int64(*)(): ctypeid_func_int64,          \
        usize(*)(): ctypeid_func_int64,              isize(*)(): ctypeid_func_int64,          \
         default: ctypeid_other)

#define ctypeidof_custom(x, ty, id) _Generic((x), ty: id, default: ctypeid_other)


// cxpanic_priv(int code, const char* msg) {
#define cxpanic_priv(code, msg) ({ \
    char buf[890] = {0}; \
    snprintf(buf, sizeof(buf)-1, "cxpanic: %d %s\n", code, msg); \
    write(2, buf, strlen(buf)); *(int*)(0)=1; \
})
#define cxunreach_priv() cxpanic_priv(0, "Unreachable")

// preadd a count param before __VA_ARGS__
#define VAARG_PREADD_NARG(...) PP_NARG(__VA_ARGS__) IFN(__VA_ARGS__)(,) __VA_ARGS__

// skip need one by one
#define VAARG_SKIP_1(a0, ...) __VA_ARGS__ // IFE(__VA_ARG__)(,) a0
#define VAARG_SKIP_2(a0, ...) VAARG_SKIP_1(__VA_ARGS__)
#define VAARG_SKIP_3(a0, ...) VAARG_SKIP_2(__VA_ARGS__)
#define VAARG_SKIP_4(a0, ...) VAARG_SKIP_3(__VA_ARGS__)
#define VAARG_SKIP_5(a0, ...) VAARG_SKIP_4(__VA_ARGS__)
#define VAARG_SKIP(N, ...) VAARG_SKIP_##N(__VA_ARGS__)

// index __VA_ARGS__, from [1-9]
#define VAARG_AT_1(X, ...) X
#define VAARG_AT_2(X, ...) VAARG_AT_1(__VA_ARGS__)
#define VAARG_AT_3(X, ...) VAARG_AT_2(__VA_ARGS__)
#define VAARG_AT_4(X, ...) VAARG_AT_3(__VA_ARGS__)
#define VAARG_AT_5(X, ...) VAARG_AT_4(__VA_ARGS__)
#define VAARG_AT_6(X, ...) VAARG_AT_5(__VA_ARGS__)
#define VAARG_AT_7(X, ...) VAARG_AT_6(__VA_ARGS__)
#define VAARG_AT_8(X, ...) VAARG_AT_7(__VA_ARGS__)
#define VAARG_AT_9(X, ...) VAARG_AT_8(__VA_ARGS__)
#define VAARG_AT_10(X, ...) VAARG_AT_9(__VA_ARGS__)
#define VAARG_AT_11(X, ...) VAARG_AT_10(__VA_ARGS__)
#define VAARG_AT_12(X, ...) VAARG_AT_11(__VA_ARGS__)
#define VAARG_AT_13(X, ...) VAARG_AT_12(__VA_ARGS__)
#define VAARG_AT_14(X, ...) VAARG_AT_13(__VA_ARGS__)
#define VAARG_AT_15(X, ...) VAARG_AT_14(__VA_ARGS__)
#define VAARG_AT_16(X, ...) VAARG_AT_15(__VA_ARGS__)
#define VAARG_AT_17(X, ...) VAARG_AT_16(__VA_ARGS__)
#define VAARG_AT_18(X, ...) VAARG_AT_17(__VA_ARGS__)
#define VAARG_AT_19(X, ...) VAARG_AT_18(__VA_ARGS__)
#define VAARG_AT_20(X, ...) VAARG_AT_19(__VA_ARGS__)
#define VAARG_AT_21(X, ...) VAARG_AT_20(__VA_ARGS__)
#define VAARG_AT_22(X, ...) VAARG_AT_21(__VA_ARGS__)
#define VAARG_AT_23(X, ...) VAARG_AT_22(__VA_ARGS__)
#define VAARG_AT_24(X, ...) VAARG_AT_23(__VA_ARGS__)
#define VAARG_AT(idx, ...) VAARG_AT_##idx(__VA_ARGS__)

#endif // _CXMACRO_P_H_
