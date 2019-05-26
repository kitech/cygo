#ifndef _NORO_PRIV_H_
#define _NORO_PRIV_H_

// std
#include <assert.h>
#include <stdlib.h>
#include <stdio.h>
#include <stdint.h>
#include <stdbool.h>
#include <errno.h>
#include <stdatomic.h>

// sys
#include <pthread.h>
#include <threads.h>
// c11 mtx_t
#include <sys/epoll.h>
#include <sys/timerfd.h>

// third
#include <coro.h>
#include <collectc/hashtable.h>
#include <collectc/array.h>

#define nilptr ((void*)NULL)

// project
#include "rxilog.h"
#include "yieldtypes.h"
#include "hookcb.h"
#include "noro_util.h"
#include "atomic.h"
#include "queue.h"
#include "chan.h"
#include "norogc.h"


// for netpoller.c
typedef struct netpoller netpoller;
netpoller* netpoller_new();
void netpoller_loop();
void netpoller_yieldfd(int fd, int ytype, void* gr);
void netpoller_use_threads();

// for goroutine
typedef struct coro_stack coro_stack;
typedef enum grstate {nostack=0, runnable, executing, waiting, finished, } grstate;
// 每个goroutine同时只能属于某一个machine
typedef struct goroutine {
    int id;
    coro_func fnproc;
    void* arg;
    coro_stack stack;
    struct GC_stack_base mystksb; // mine for GC
    coro_context coctx;
    coro_context coctx0;
    grstate state;
    bool isresume;
    void* hcelem;
    int pkstate;
    struct GC_stack_base* stksb; // machine's
    void* gchandle;
    int  mcid;
    void* savefrm; // upper frame
    void* myfrm; // my frame when yield
} goroutine;

// processor callbacks, impl in noro.c
extern int noro_processor_yield(int fd, int ytype);
extern bool noro_in_processor();
extern void noro_processor_resume_some(void* gr_);
extern goroutine* noro_goroutine_getcur();

#endif

