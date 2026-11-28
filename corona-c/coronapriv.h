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
#include <signal.h>

// sys
#include <time.h>
// #include <pthread.h>
// #include <threads.h> // c11 mtx_t
#if defined(__linux__)
#include <sys/epoll.h>
#include <sys/timerfd.h>
#endif

// third
#include "coro.h"
//#include "collectc/array.h"
//#include "collectc/hashtable.h"
//#include "collectc/treetable.h"
//#include "collectc/pqueue.h"
//#include <collect_cds.h>

typedef struct array_s Array;
typedef struct hashtable_s HashTable;

#ifndef nilptr
#define nilptr ((void*)NULL)
#endif
#define invalidptr ((void*)1234567)

// project
#include "datstu.h"
#include "cxtypedefs.h"
#include "rxilog.h"
#include "yieldtypes.h"
#include "iohookcb.h"
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
void netpoller_yieldfd(int64_t fd, int ytype, fiber* gr);
void netpoller_use_threads();
// const char* netpoller_name();

// 取消该 fiber 的全部未决 netpoller 事件（destroy 兜底 / poll-select 唤
// 醒后的兄弟清理）。ev/uv 后端内部经各自线程桥转到 loop 线程执行。
void netpoller_cancelfiber(fiber* gr);

// netpoller 未决事件登记（datstu.c 提供无锁操作，三后端共享）。
// evdata 的首成员必须是 npreg_hdr；注册方 add（fiber 自己的 machine
// 线程），回调先 claim 再触碰字段、收尾 done，cancelfiber 按 fiber drain。
// 线程纪律与状态机见 datstu.c 注释。全程无锁。
typedef struct npreg_hdr {
    void* npnext;   // 仅该 fiber 的 machine 线程读写
    int npstate;    // 仅原子操作：见 NPREG_* 状态字
} npreg_hdr;
#define NPREG_REGD     0   // 已注册（事件已武装、未触发）
#define NPREG_CLAIMED  1   // 回调已认领（回调负责 event_free，不碰链表）
#define NPREG_DONE     2   // 回调已收尾（drain 可回收 evdata）
#define NPREG_DRAINING 4   // drain 已接管归属（回调认领失败即放弃）
#define NPREG_DYING    8   // fiber 正在销毁：在途回调跳过唤醒

void crn_npreg_add(fiber* gr, void* d);
void crn_npreg_undo(fiber* gr, void* d);
int  crn_npreg_claim(void* d);
void crn_npreg_done(void* d);
int  crn_npreg_drain(fiber* gr, void (*free_fn)(void* d));
// 给 ev/uv（注册/释放都在 loop 线程）的收集式 drain：见 datstu.c 注释
int  crn_npreg_take(fiber* gr, void** arr, int cap);

// ev/uv 后端用：机器线程创建令牌入列并随 job 发往 loop，loop 把自己的
// watcher 对象挂到 obj 上；唤醒/取消都只碰令牌的原子状态字。
// kind: 令牌类别（io/timer/dns），取消时 loop 侧据此释放 obj。
typedef struct np_token {
    npreg_hdr np;
    int kind;
    void* obj;   // loop 线程侧对象（evwaiter/evsleep/evdns_req/uvwaiter/...）
} np_token;
#define NPT_IO    1
#define NPT_TIMER 2
#define NPT_DNS   3

extern int crn_goid();

// for fiber
typedef struct coro_stack coro_stack;
/* corona reserves this many bytes at the low end of every fiber stack and
 * mprotects them PROT_READ (see crn_fiber_new2). libco must not place its
 * saved-register block there. */
#define CRN_FIBER_GUARD_SIZE 4096
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
    int wake_pending; // cross-thread resume raced with a still-running fiber
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
    void* nphead;    // netpoller 未决事件链头（npreg_hdr；单写者=本fiber的machine线程）
};

// procer callbacks, impl in corona.c
extern int crn_procer_yield(int64_t fd, int ytype);
extern int crn_procer_yield_multi(int ytype, int nfds, const int64_t fds[], const int ytypes[]);
extern bool crn_in_procer();
extern void crn_procer_resume_one(void* gr_, int ytype, int grid, int mcid);
extern fiber* crn_fiber_getcur();
extern void* crn_fiber_getspec(void* spec);
extern void crn_fiber_setspec(void* spec, void* val);
extern int crn_sleepms_nohook(int val);

/* Set to 1 by the application's signal handler (SIGINT/SIGTERM/...).
 * Sleep hooks return -1/EINTR while it is non-zero.
 * Not auto-cleared: call crn_clear_signal_interrupt() after handling. */
extern volatile sig_atomic_t crn_signal_interrupt_flag;
void crn_clear_signal_interrupt(void);

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
