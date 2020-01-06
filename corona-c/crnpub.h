#ifndef _CRN_PUB_H_
#define _CRN_PUB_H_

typedef struct corona corona;

typedef struct crn_inner_stats {
    int mch_totcnt;
    int mch_actcnt;
    int fiber_totcnt;
    int fiber_actcnt;
    int fiber_totmem;
    int maxstksz;
} crn_inner_stats;

corona* crn_init_and_wait_done();

extern int crn_goid();
extern int crn_post(void(*fn)(void*arg), void*arg);
extern void crn_sched();

extern void crn_lock_osthread();
extern void crn_get_stats(crn_inner_stats* st);

#endif

