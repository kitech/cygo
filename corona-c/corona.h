
#ifndef _CORONA_H_
#define _CORONA_H_

#include <stddef.h>

typedef struct corona corona;
// typedef struct fiber fiber;

corona* crn_get();
corona* crn_new();
void crn_init(corona* nr);
void crn_destroy(corona* lnr);
void crn_wait_init_done(corona* nr);
corona* crn_init_and_wait_done();
// 在crn_init*之前调用，设置线程创建成功回调通知，做线程初始化
void* crn_set_thread_createcb(void(*fn)(void*arg), void* arg);

int crn_post(void(*fn)(void*arg), void*arg);
int crn_post_sized(void(*fn)(void*arg), void*arg, size_t stksz);
void crn_sched();
int crn_num_fibers();
int crn_goid();
// fiber* crn_fiber_getcur();
int crn_fiber_stackaddr_cur(void** addr, size_t *size);

#endif
