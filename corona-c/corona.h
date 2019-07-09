
#ifndef _CORONA_H_
#define _CORONA_H_


typedef struct corona corona;

corona* crn_get();
corona* crn_new();
void crn_init(corona* nr);
void crn_destroy(corona* lnr);
void crn_wait_init_done(corona* nr);
corona* crn_init_and_wait_done();
// 在crn_init*之前调用，设置线程创建成功回调通知，做线程初始化
void* crn_set_thread_createcb(void(*fn)(void*arg), void* arg);

int crn_post(void(*fn)(void*arg), void*arg);
void crn_sched();
int crn_num_fibers();

#endif
