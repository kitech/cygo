#ifndef _CRN_PUB_H_
#define _CRN_PUB_H_

#include <stddef.h>

typedef struct corona corona;

typedef struct crn_inner_stats {
    int mch_totcnt;
    int mch_actcnt;
    int fiber_totcnt;
    int fiber_actcnt;
    int fiber_totmem;
    int maxstksz;
} crn_inner_stats;

typedef struct crn_fiber_info {
    int id;            /* fiber id */
    int mcid;          /* machine id */
    int state;         /* grstate: 0 nostack,1 runnable,2 executing,3 waiting,4 finished */
    int pkreason;      /* last yield type */
    size_t stksz;      /* total stack size */
    size_t used_stksz; /* stack high-water mark */
} crn_fiber_info;

/*
 * Snapshot of currently registered fibers (grs maps; not-yet-dispatched
 * ngrs entries are excluded).  Fills at most cap entries into out.
 * Returns the total number found (possibly > cap), or <0 on bad arguments.
 * out may be NULL when cap == 0.  This is a best-effort, non-atomic view
 * across machines; it is not async-signal-safe.
 */
extern int crn_get_fibers(crn_fiber_info* out, int cap);

corona* crn_init_and_wait_done();

extern int crn_log_set_level(int);
extern int crn_goid();
extern int crn_post(void(*fn)(void*arg), void*arg);
extern void crn_sched();

extern void crn_lock_osthread();
extern void crn_get_stats(crn_inner_stats* st);
extern int crn_num_fibers(void);

#endif
