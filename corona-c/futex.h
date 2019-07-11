#ifndef _FUTEX_H_
#define _FUTEX_H_

#include <pthread.h>
#include <time.h>

typedef pthread_mutex_t pmutex_t;
typedef pthread_mutexattr_t pmutexattr_t;
typedef pthread_cond_t pcond_t;
typedef pthread_condattr_t pcondattr_t;

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

//
typedef struct fiber fiber;
typedef struct crnqueue crnqueue;
typedef struct crn_mutex {
    pthread_mutex_t lock;
    crnqueue* waitq; // fiber*
    fiber* holder; // fiber*
} crn_mutex;
typedef pthread_mutexattr_t crn_mutexattr;
typedef struct crn_cond {
    pthread_cond_t cd;
    crn_mutex* mutex;
    crnqueue* waitq; // fiber*
    fiber* holder; // fiber*
} crn_cond;
typedef pthread_condattr_t crn_condattr;

crn_mutex* crn_mutex_new();
int crn_mutex_lock(crn_mutex *mutex);
int crn_mutex_trylock(crn_mutex *mutex);
int crn_mutex_unlock(crn_mutex *mutex);
int crn_mutex_destroy(crn_mutex *mutex);

crn_cond* crn_cond_new(crn_mutex* mutex);
int crn_cond_timedwait(crn_cond * cond, const struct timespec * abstime);
int crn_cond_wait(crn_cond * cond);
int crn_cond_broadcast(crn_cond *cond);
int crn_cond_signal(crn_cond *cond);
int crn_cond_destroy(crn_cond *cond);


#endif

