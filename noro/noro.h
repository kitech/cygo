
#ifndef _NORO_H_
#define _NORO_H_


typedef struct noro noro;

noro* noro_get();
noro* noro_new();
void noro_init(noro* nr);
void noro_destroy(noro* lnr);
void noro_wait_init_done(noro* nr);
noro* noro_init_and_wait_done();
// 在noro_init*之前调用，设置线程创建成功回调通知，做线程初始化
void* noro_set_thread_createcb(void(*fn)(void*arg), void* arg);

void noro_post(void(*fn)(void*arg), void*arg);
void noro_sched();

#endif
