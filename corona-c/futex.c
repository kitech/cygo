#include <assert.h>
#include <pthread.h>

#include "iohook.h"
#include "futex.h"
#include "coronagc.h"
#include "coronapriv.h"

extern void crn_pre_gclock_proc(const char* funcname);
extern void crn_post_gclock_proc(const char* funcname);

// for some internal use, cannot let thread yield
int pmutex_lock(pmutex_t *mutex)
{
    if (pmutex_lock_f == nilptr) { initHook(); }
    assert(pmutex_lock_f != nilptr);
    crn_pre_gclock_proc(__func__);
    int rv = pmutex_lock_f(mutex);
    crn_post_gclock_proc(__func__);
    assert(rv==0);
    return rv;
}
int pmutex_trylock(pmutex_t *mutex)
{
    crn_pre_gclock_proc(__func__);
    int rv = pmutex_trylock_f(mutex);
    crn_post_gclock_proc(__func__);
    return rv;
}
int pmutex_unlock(pmutex_t *mutex)
{
    if (pmutex_unlock_f == nilptr) { initHook(); }
    assert(pmutex_unlock_f != nilptr);
    crn_pre_gclock_proc(__func__);
    int rv = pmutex_unlock_f(mutex);
    crn_post_gclock_proc(__func__);
    return rv;
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
    if (pcond_timedwait_f == nilptr) { initHook(); }
    return pcond_timedwait_f(cond, mutex, abstime);
}
int pcond_wait(pcond_t * cond, pmutex_t * mutex)
{
    if (pcond_wait_f == nilptr) { initHook(); }
    int rv = pcond_wait_f(cond, mutex);
    assert(rv==0);
    return rv;
}
int pcond_broadcast(pcond_t *cond)
{
    if (pcond_broadcast_f == nilptr) { initHook(); }
    return pcond_broadcast_f(cond);
}
int pcond_signal(pcond_t *cond)
{
    if (pcond_signal_f == nilptr) { initHook(); }
    return pcond_signal_f(cond);
}
int pcond_destroy(pcond_t *cond)
{
    return pthread_cond_destroy(cond);
}
int pcond_init(pcond_t *cond, const pcondattr_t *attr)
{
    return pthread_cond_init(cond, attr);
}

// for pub usage, will yield but not block thread/fiber
static void crn_mutex_finalzier(void* mu) {
}
crn_mutex* crn_mutex_new() {
    crn_mutex* mup = (crn_mutex*)crn_gc_malloc(sizeof(crn_mutex));
    int rv = pthread_mutex_init(&mup->lock, 0);
    assert(rv == 0);
    rv = pthread_mutex_init(&mup->qmu, 0);
    assert(rv == 0);
    mup->waitq = crnqueue_new();
    return mup;
}
int crn_mutex_lock(crn_mutex *mutex)
{
    fiber* mygr = crn_fiber_getcur();
    assert(mygr != nilptr);

    // check+enqueue under qmu: otherwise unlock can clear the holder and poll
    // the (still empty) waitq in the window before we enqueue -> lost wakeup
    pthread_mutex_lock(&mutex->qmu);
    bool bv = atomic_casptr((void**)&mutex->holder, nilptr, mygr);
    if (!bv) {
        int rv = crnqueue_enqueue(mutex->waitq, mygr);
        assert(rv == CC_OK);
    }
    pthread_mutex_unlock(&mutex->qmu);
    if (!bv) {
        crn_procer_yield(-1, YIELD_TYPE_LOCK);
    }
    crn_pre_gclock_proc(__func__);
    int rv = pthread_mutex_lock(&mutex->lock);
    assert(rv == 0);
    crn_post_gclock_proc(__func__);
    return rv;
}
int crn_mutex_trylock(crn_mutex *mutex)
{
    crn_pre_gclock_proc(__func__);
    int rv = pthread_mutex_trylock(&mutex->lock);
    crn_post_gclock_proc(__func__);
    return rv;
}
int crn_mutex_unlock(crn_mutex *mutex)
{
    fiber* mygr = crn_fiber_getcur();
    assert(mygr != nilptr);

    crn_pre_gclock_proc(__func__);
    // clear holder + pick a waiter under qmu (atomic w.r.t. lock's enqueue),
    // then release the real mutex before resuming: the woken fiber may run
    // immediately on this thread and lock mutex->lock itself
    void* wtgr = nilptr;
    pthread_mutex_lock(&mutex->qmu);
    atomic_setptr((void**)&mutex->holder, nilptr);
    int rv2 = crnqueue_poll(mutex->waitq, &wtgr);
    pthread_mutex_unlock(&mutex->qmu);
    // empty queue is normal: nobody waiting
    assert(rv2 == CC_OK || rv2 == CC_ERR_OUT_OF_RANGE);
    int rv = pthread_mutex_unlock(&mutex->lock);
    crn_post_gclock_proc(__func__);
    assert(rv == 0);

    if (wtgr != nilptr) {
        // must pass the waker fiber's own id/mcid, not the unlocker's;
        // resume_one validates grid and silently drops otherwise -> deadlock
        fiber* wgr = (fiber*)wtgr;
        atomic_setptr((void**)&mutex->holder, wgr); // handoff reservation
        crn_procer_resume_one(wtgr, 0, wgr->id, wgr->mcid);
    }
    return rv;
}
int crn_mutex_destroy(crn_mutex *mutex)
{
    int rv = pthread_mutex_destroy(&mutex->lock);
    crn_gc_free(mutex);
    return rv;
}

static void crn_cond_finalizer(void* cd) {
}
crn_cond* crn_cond_new(crn_mutex* mutex) {
    crn_cond* cdp = (crn_cond*)crn_gc_malloc(sizeof(crn_cond));
    int rv = pthread_cond_init(&cdp->cd, 0);
    assert(rv == 0);
    assert(mutex != 0);
    cdp->mutex = mutex;
    cdp->waitq = crnqueue_new();
    return cdp;
}
int crn_cond_timedwait(crn_cond * cond, const struct timespec * abstime)
{
    return pthread_cond_timedwait(&cond->cd, &cond->mutex->lock, abstime);
}
int crn_cond_wait(crn_cond * cond)
{
    return pthread_cond_wait(&cond->cd, &cond->mutex->lock);
}
int crn_cond_broadcast(crn_cond *cond)
{
    return pthread_cond_broadcast(&cond->cd);
}
int crn_cond_signal(crn_cond *cond)
{
    return pthread_cond_signal(&cond->cd);
}
int crn_cond_destroy(crn_cond *cond)
{
    int rv = pthread_cond_destroy(&cond->cd);
    crn_gc_free(cond);
    return rv;
}
