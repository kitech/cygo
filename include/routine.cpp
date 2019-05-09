#include <gc/gc_cpp.h>

#include "libgo.h"

co::AsyncCoroutinePool * pPool = 0;

extern "C"
void cxrt_init_routine_env() {
    pPool = co::AsyncCoroutinePool::Create();
    pPool->InitCoroutinePool(512);
    pPool->Start(2,5);
}

void foo()
{
    printf("do some block things in co::AsyncCoroutinePool.\n");
}

void done()
{
    printf("done.\n");
}

extern "C"
void cxrt_routine_post(void (*f)(void*), void*arg) {
    auto fo = std::bind(f, arg);
    pPool->Post(fo, &done);
}

extern "C"
void* cxrt_chan_new(int sz) {
    auto ch = ::new (UseGC) co_chan<void*>(sz);
    return ch;
}

extern "C"
void cxrt_chan_send(void*vch, void*arg) {
    co_chan<void*>* ch = static_cast<co_chan<void*>*>(vch);
    (*ch) << arg;
}
extern "C"
void* cxrt_chan_recv(void*vch) {
    co_chan<void*>* ch = static_cast<co_chan<void*>*>(vch);
    co_chan<void*>& ch2 = (*ch);

    void*ret = 0;
    ch2 >> ret;
    return ret;
}

