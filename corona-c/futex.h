#ifndef _FUTEX_H_
#define _FUTEX_H_

#include <pthread.h>
#include <time.h>

typedef pthread_mutex_t pmutex_t;
typedef pthread_mutexattr_t pmutexattr_t;
typedef pthread_cond_t pcond_t;
typedef pthread_cond_t pcondattr_t;

int pmutex_lock(pmutex_t *mutex);
int pmutex_trylock(pmutex_t *mutex);
int pmutex_unlock(pmutex_t *mutex);
int pmutex_init(pmutex_t *mutex, const pmutexattr_t *attr);
int pmutex_destroy(pmutex_t *mutex);

int pcond_timedwait(pcond_t * cond, pmutex_t * mutex, const struct timespec * abstime);
int pcond_wait(pcond_t * cond, pmutex_t * mutex);
int pcond_broadcast(pcond_t *cond);
int pcond_signal(pcond_t *cond);
int pcond_destroy(pcond_t *cond);
int pcond_init(pcond_t *cond, const pcondattr_t *attr);


#endif

