
#include "libgo.h"

co::AsyncCoroutinePool * pPool = 0;

extern "C"
void cxrt_init_routine_env() {
    pPool = co::AsyncCoroutinePool::Create();
    pPool->InitCoroutinePool(512);
    pPool->Start(2,6);
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
void cxrt_routine_post(void (*f)()) {
    pPool->Post(f, &done);
}

