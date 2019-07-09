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
// #include <pthread.h>
// #include <threads.h> // c11 mtx_t
#include <sys/epoll.h>
#include <sys/timerfd.h>

// third
#include "coro.h"
#include "collectc/hashtable.h"
#include "collectc/array.h"

#define nilptr ((void*)NULL)
#define invlidptr ((void*)1234567)

// project
#include "datstu.h"
#include "rxilog.h"
#include "yieldtypes.h"
#include "hookcb.h"
#include "futex.h"
#include "atomic.h"
#include "corona_util.h"
#include "szqueue.h"
#include "chan.h"
#include "coronagc.h"


typedef struct fiber fiber;

// for netpoller.c
typedef struct netpoller netpoller;
netpoller* netpoller_new();
void netpoller_loop();
void netpoller_yieldfd(long fd, int ytype, fiber* gr);
void netpoller_use_threads();

// for fiber
typedef struct coro_stack coro_stack;
typedef enum grstate {nostack=0, runnable, executing, waiting, finished, } grstate;
extern const char* grstate2str(grstate s);
// 每个fiber 同时只能属于某一个machine
struct fiber {
    int id;
    coro_func fnproc;
    void* arg;
    coro_stack stack;
    struct GC_stack_base mystksb; // mine for GC
    coro_context coctx;
    coro_context *coctx0; // ref to machine.coctx0
    grstate state;
    bool isresume;
    void* hcelem;
    pmutex_t* hclock; // hchan.lock
    int pkreason;
    fiber* wokeby; //
    void* wokehc; // hchan*
    int wokecase; // caseSend/caseRecv
    struct GC_stack_base* stksb; // machine's
    void* gchandle;
    int  mcid;
    void* savefrm; // upper frame
    void* myfrm; // my frame when yield
    // really crnmap*
    HashTable* specifics; // like thread specific // int* => void*, value can pass to free()
};

// procer callbacks, impl in corona.c
extern int crn_procer_yield(long fd, int ytype);
extern int crn_procer_yield_multi(int ytype, int nfds, long fds[], int ytypes[]);
extern bool crn_in_procer();
extern void crn_procer_resume_one(void* gr_, int ytype, int grid, int mcid);
extern fiber* crn_fiber_getcur();
extern void* crn_fiber_getspec(void* spec);
extern void crn_fiber_setspec(void* spec, void* val);

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

