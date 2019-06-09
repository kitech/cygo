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
#include "coro.h"
#include "collectc/hashtable.h"
#include "collectc/array.h"

#define nilptr ((void*)NULL)
#define invlidptr ((void*)1234567)

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
void netpoller_yieldfd(long fd, int ytype, void* gr);
void netpoller_use_threads();

// for goroutine
typedef struct coro_stack coro_stack;
typedef enum grstate {nostack=0, runnable, executing, waiting, finished, } grstate;
// 每个goroutine同时只能属于某一个machine
typedef struct goroutine goroutine;
struct goroutine {
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
    mtx_t* hclock; // hchan.lock
    int pkreason;
    goroutine* wokeby; //
    void* wokehc; // hchan*
    int wokecase; // caseSend/caseRecv
    struct GC_stack_base* stksb; // machine's
    void* gchandle;
    int  mcid;
    void* savefrm; // upper frame
    void* myfrm; // my frame when yield
};

// processor callbacks, impl in noro.c
extern int noro_processor_yield(long fd, int ytype);
extern int noro_processor_yield_multi(int ytype, int nfds, long fds[], int ytypes[]);
extern bool noro_in_processor();
extern void noro_processor_resume_some(void* gr_, int ytype);
extern goroutine* noro_goroutine_getcur();

extern void loglock();
extern void logunlock();


#define YIELD_NORM_NS 1000
#define YIELD_CHAN_NS 1001

// hselect cases
enum {
      caseNil = 0,
      caseRecv,
      caseSend,
      caseDefault,
      caseClose, // nothing but for special
};

#endif

