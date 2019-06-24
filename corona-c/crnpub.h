#ifndef _CRN_PUB_H_
#define _CRN_PUB_H_

typedef struct corona corona;

corona* crn_init_and_wait_done();

extern int crn_get_goid();
extern void crn_post(void(*fn)(void*arg), void*arg);
extern void crn_sched();

#endif
