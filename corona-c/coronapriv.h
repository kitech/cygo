#ifndef _NORO_PRIV_H_
#define _NORO_PRIV_H_

// std
#include <stddef.h>
#include <assert.h>
#include <stdlib.h>
#include <stdio.h>
#include <stdint.h>
#include <stdbool.h>
#include <errno.h>
#include <stdatomic.h>

// sys
#include <time.h>
// #include <pthread.h>
// #include <threads.h> // c11 mtx_t
#ifdef __APPLE__
#elif _WIN32
#else
#include <sys/epoll.h>
#include <sys/timerfd.h>
#endif

// third
#include "coro.h"
#include "collectc/array.h"
#include "collectc/hashtable.h"
#include "collectc/treetable.h"
#include "collectc/pqueue.h"

#ifndef nilptr
#define nilptr ((void*)NULL)
#endif
#define invlidptr ((void*)1234567)

// project
#include "datstu.h"
#include "cxtypedefs.h"
#include "rxilog.h"
#include "yieldtypes.h"
#include "hookcb.h"
#include "futex.h"
#include "atomic.h"
// #include "rxilog.h"
#include "corona_util.h"
#include "szqueue.h"
#include "chan.h"
#include "coronagc.h"
#include "netpoller.h"


typedef struct fiber fiber;

// for netpoller.c
typedef struct netpoller netpoller;
netpoller* netpoller_new();
void netpoller_loop();
void netpoller_yieldfd(long fd, int ytype, fiber* gr);
void netpoller_use_threads();
// const char* netpoller_name();

// for fiber
typedef struct coro_stack coro_stack;
typedef enum grstate {nostack=0, runnable, executing, waiting, finished, } grstate;
extern const char* grstate2str(grstate s);
// 每个fiber 同时只能属于某一个machine
struct fiber {
    int id;
    int  mcid;
    int usemmap;
    int libcmalloc;
    size_t guardsize;
    coro_func fnproc;
    void* arg;
    coro_stack stack;
    void* stkptr;
    size_t stksz;
    void* stkmid;
    struct GC_stack_base mystksb; // mine for GC
    void* sig_regi_ticket; // sigsegv_register
    coro_context coctx;
    char overflowed[999]; // for upper coctx
    coro_context *coctx0; // ref to machine.coctx0
    grstate state;
    int isresume;
    pmutex_t* hclock; // hchan.lock
    int pkreason;
    struct timeval pktime;
    struct GC_stack_base* stksb; // machine's
    void* gchandle;
    void* savefrm; // upper frame
    void* myfrm; // my frame when yield
    // this should not access multiple thread, so just use non-lock hashtable
    HashTable* specifics; // like thread specific // int* => void*, value can pass to free()
    int lock_osthr; // lock os thread
    void* used_stkbottom;
    int used_stksz;  // = used_stkbottom - stack.sptr(stktop)
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
