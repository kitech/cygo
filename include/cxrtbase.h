#ifndef _CXRT_BASE_H_
#define _CXRT_BASE_H_

#include <stdio.h>
#include <stdlib.h>
#include <pthread.h>
#include <stdint.h>
#include <gc/gc.h> // must put after <pthread.h>
#include <gc/cord.h>

typedef uint8_t bool;
typedef uint8_t byte;
typedef uint8_t uint8;
typedef int8_t int8;
typedef uint16_t uint16;
typedef int16_t int16;
typedef uint32_t uint32;
typedef int32_t int32;
typedef uint64_t uint64;
typedef int64_t int64;
typedef float float32;
typedef double float64;
typedef uintptr_t uintptr;

typedef struct {
    char* data;
    int len;
} string;

void println(const char* fmt, ...);

// TODO
#define gogorun

extern void cxrt_init_env();
extern void cxrt_routine_post(void (*f)(void*), void*arg);
extern void* cxrt_chan_new(int sz);
extern void cxrt_chan_send(void*ch, void*arg);
extern void* cxrt_chan_recv(void*arg);

#include <sys/types.h>
extern pid_t gettid();

#endif
