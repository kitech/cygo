#include <pthread.h>

#include "hook.h"
#include "futex.h"

// for some internal use, cannot let thread yield
int pmutex_lock(pmutex_t *mutex)
{
    return pmutex_lock_f(mutex);
}
int pmutex_trylock(pmutex_t *mutex)
{
    return pmutex_trylock_f(mutex);
}
int pmutex_unlock(pmutex_t *mutex)
{
    return pmutex_unlock_f(mutex);
}
int pmutex_init(pmutex_t *mutex, const pmutexattr_t *attr)
{
    return pthread_mutex_init(mutex, attr);
}
int pmutex_destroy(pmutex_t *mutex)
{
    return pthread_mutex_destroy(mutex);
}

int pcond_timedwait(pcond_t * cond, pmutex_t * mutex,
                   const struct timespec * abstime)
{
    return pcond_timedwait_f(cond, mutex, abstime);
}
int pcond_wait(pcond_t * cond, pmutex_t * mutex)
{
    return pcond_wait_f(cond, mutex);
}
int pcond_broadcast(pcond_t *cond)
{
    return pcond_broadcast_f(cond);
}
int pcond_signal(pcond_t *cond)
{
    return pcond_signal_f(cond);
}
int pcond_destroy(pcond_t *cond)
{
    return pthread_mutex_destroy(cond);
}
int pcond_init(pcond_t *cond, const pcondattr_t *attr)
{
    return pthread_cond_init(cond, attr);
}
