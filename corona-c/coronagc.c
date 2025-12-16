#include <assert.h>
#include <dlfcn.h>
#include <stdlib.h>
#include <stdio.h>
#include <stdbool.h>
#include <string.h>
#include <unistd.h>

#include "coronagc.h"
#include "coronapriv.h"

void* crn_raw_malloc(size_t size) {
     return calloc(1, size);
}
/* void* crn_raw_realloc(void* ptr, size_t size) { */
/*     return realloc(ptr, size); */
/* } */
void crn_raw_free(void* ptr) {
    return free(ptr);
}
/* void* crn_raw_calloc(size_t n, size_t size) { */
/*     return calloc(n, size); */
/* } */

///////

gcstates crn_gc_states = {0};

static void crn_pre_gclock_void(const char* funcname) {}
static void crn_post_gclock_void(const char* funcname) {}

void (*crn_pre_gclock_fn)(const char* funcname) = crn_pre_gclock_void;
void (*crn_post_gclock_fn)(const char* funcname) = crn_post_gclock_void;

int crn_gc_ready() { return crn_pre_gclock_fn != 0; }

#ifdef USE_BDWGC

static void crn_pre_gclock(const char* funcname) {
    int rv = atomic_addint(&crn_gc_states.ingclock, 1);
    assert(rv >= 0);
    // if (crn_pre_gclock_fn == 0) { return ; } // temporary test
    assert(crn_pre_gclock_fn != 0);
    crn_pre_gclock_fn(funcname);
}
static void crn_post_gclock(const char* funcname) {
    int rv = atomic_addint(&crn_gc_states.ingclock, -1);
    assert(rv>0);
    // if (crn_post_gclock_fn == 0) { return ; }
    assert(crn_post_gclock_fn != 0);
    crn_post_gclock_fn(funcname);
}
static void crn_gc_finalizer(void*ptr, void*clientdata) {
    printf("ptr dtor %p\n", ptr);
}

int crn_gc_deadlock_detect1() {
    if (atomic_getint(&crn_gc_states.stopworld)==1) {
        lwarn("stopworld little danger %d\n", 0);
        int waitcnt = 0;
        int (*usleep_f)(long) = dlsym(RTLD_NEXT, "usleep");
        assert(usleep_f != NULL);
        // extern void (*usleep_f)(int);
        while(atomic_getint(&crn_gc_states.stopworld)==1) {
            usleep_f(1234); // 1ms
            waitcnt++;
        }
        lwarn("stopworld little danger avoided %d\n", waitcnt);
    }
    return 0;
}

void* crn_gc_malloc(size_t size) {
    assert(atomic_getint(&crn_gc_states.stopworld2)==0); // or deadlock
    // linux only deadlock
#ifdef __APPLE__
#else
    crn_gc_deadlock_detect1();
#endif

    crn_pre_gclock(__func__);
    void* ptr = GC_MALLOC(size);
    crn_post_gclock(__func__);
    memset(ptr, 0, size);
    // GC_register_finalizer(ptr, crn_gc_finalizer, 0, 0, 0);
    return ptr;
}
void* crn_gc_realloc(void* ptr, size_t size) {
    assert(atomic_getint(&crn_gc_states.stopworld2)==0); // or deadlock
    crn_pre_gclock(__func__);
    void* newptr = GC_REALLOC(ptr, size);
    crn_post_gclock(__func__);
    return newptr;
}
static void* crn_gc_free_block(void* ptr) {
    // crn_pre_gclock();
    GC_FREE(ptr);
    // crn_post_gclock();
    return 0;
}
// GC_clear_fl_marks loops https://www.hpl.hp.com/hosted/linux/mail-archives/gc/2008-March/002176.html
// The only way I know for this to happen is if you call GC_free() on thesame object twice.
static int __we_take_free_owner = 0;
void crn_gc_free(void* ptr) {
    if (ptr == 0) return;
    crn_pre_gclock(__func__);
    if (__we_take_free_owner == 1) {
        GC_FREE(ptr);
        // GC_do_blocking(crn_gc_free_block, ptr);
    }else{}
    crn_post_gclock(__func__);
}
void crn_gc_free_uncollectable(void* ptr) {
    crn_pre_gclock(__func__);
    GC_FREE(ptr);
    crn_post_gclock(__func__);
}
void* crn_gc_calloc(size_t n, size_t size) {
    // assert(atomic_getint(&crn_gc_states.incollect)==0); // or deadlock
    crn_pre_gclock(__func__);
    void* ptr = GC_MALLOC(n*size);
    crn_post_gclock(__func__);
    memset(ptr, 0, n*size);
    return ptr;
}
void* crn_gc_malloc_uncollectable(size_t size) {
    // assert(atomic_getint(&crn_gc_states.incollect)==0); // or deadlock
    crn_pre_gclock(__func__);
    void* ptr = GC_MALLOC_UNCOLLECTABLE(size);
    crn_post_gclock(__func__);
    memset(ptr, 0, size);
    return ptr;
}

void crn_call_with_alloc_lock(void*(*fnptr)(void* arg1), void* arg) {
    crn_pre_gclock(__func__);
    #ifdef __APPLE__
    // GC_call_with_alloc_lock(fnptr, arg);
    // GC_do_blocking(fnptr, arg);
    GC_call_with_gc_active(fnptr, arg);
    #else
    #endif
    crn_post_gclock(__func__);
}

static void crn_finalizer_fwd(void* ptr, void* fnptr) {
    ((void (*)(void*))fnptr)(ptr);
}
void crn_set_finalizer(void* ptr, void(*ufin)(void* ptr)) {
    assert(ptr!=nilptr);
    crn_pre_gclock(__func__);
    if (ufin == NULL) {
        GC_REGISTER_FINALIZER(ptr, NULL, NULL, NULL, NULL);
    }else{
        GC_REGISTER_FINALIZER(ptr, crn_finalizer_fwd, ufin, NULL, NULL);
    }
    crn_post_gclock(__func__);
}

void crn_gc_set_nprocs(int n) {
    char strn[32] = {0};
    snprintf(strn, sizeof(strn)-1, "%d", n);
    setenv("GC_NPROCS", strn, false);
    // bdwgc default is NCPU-1
}

const char* crn_gc_event_name(GC_EventType evty) {
    switch (evty) {
    case GC_EVENT_START: /* COLLECTION */
        return "clct-start";
    case GC_EVENT_MARK_START:
        return "mark-start";
    case GC_EVENT_MARK_END:
        return "mark-end";
    case GC_EVENT_RECLAIM_START:
        return "reclaim-start";
    case GC_EVENT_RECLAIM_END:
        return "reclaim-end";
    case GC_EVENT_END: /* COLLECTION */
        return "clct-end";
    case GC_EVENT_PRE_STOP_WORLD: /* STOPWORLD_BEGIN */
        return "pre-stopworld";
    case GC_EVENT_POST_STOP_WORLD: /* STOPWORLD_END */
        return "post-stopworld";
    case GC_EVENT_PRE_START_WORLD: /* STARTWORLD_BEGIN */
        return "pre-startworld";
    case GC_EVENT_POST_START_WORLD: /* STARTWORLD_END */
        return "post-startworld";
    case GC_EVENT_THREAD_SUSPENDED:
        return "thread-suspend";
    case GC_EVENT_THREAD_UNSUSPENDED:
        return "thread-unsuspend";
    }
    assert(1==2);
}
#endif
