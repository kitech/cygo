#include <assert.h>
#include <stdlib.h>
#include <stdio.h>
#include <stdbool.h>

#include "coronagc.h"

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

#ifdef USE_BDWGC

void (*crn_pre_gclock_fn)(const char* funcname) = 0;
void (*crn_post_gclock_fn)(const char* funcname) = 0;

static void crn_pre_gclock(const char* funcname) {
    assert(crn_pre_gclock_fn != 0);
    crn_pre_gclock_fn(funcname);
}
static void crn_post_gclock(const char* funcname) {
    assert(crn_post_gclock_fn != 0);
    crn_post_gclock_fn(funcname);
}
static void crn_gc_finalizer(void*ptr, void*clientdata) {
    printf("ptr dtor %p\n", ptr);
}
void* crn_gc_malloc(size_t size) {
    crn_pre_gclock(__func__);
    void* ptr = GC_MALLOC(size);
    crn_post_gclock(__func__);
    // GC_register_finalizer(ptr, crn_gc_finalizer, 0, 0, 0);
    return ptr;
}
void* crn_gc_realloc(void* ptr, size_t size) {
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
void crn_gc_free(void* ptr) {
    if (ptr == 0) return;
    crn_pre_gclock(__func__);
    // GC_FREE(ptr);
    GC_do_blocking(crn_gc_free_block, ptr);
    crn_post_gclock(__func__);
}
void crn_gc_free2(void* ptr) {
    crn_pre_gclock(__func__);
    GC_FREE(ptr);
    crn_post_gclock(__func__);
}
void* crn_gc_calloc(size_t n, size_t size) {
    crn_pre_gclock(__func__);
    void* ptr = GC_MALLOC(n*size);
    crn_post_gclock(__func__);
    return ptr;
}
void* crn_gc_malloc_uncollectable(size_t size) {
    crn_pre_gclock(__func__);
    void* ptr = GC_MALLOC_UNCOLLECTABLE(size);
    crn_post_gclock(__func__);
    return ptr;
}

void crn_call_with_alloc_lock(void*(*fnptr)(void* arg1), void* arg) {
    crn_pre_gclock(__func__);
    // GC_call_with_alloc_lock(fnptr, arg);
    // GC_do_blocking(fnptr, arg);
    GC_call_with_gc_active(fnptr, arg);
    crn_post_gclock(__func__);
}

static void crn_finalizer_fwd(void* ptr, void* fnptr) {
    ((void (*)(void*))fnptr)(ptr);
}
void crn_set_finalizer(void* ptr, void(*ufin)(void* ptr)) {
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
        return "clctstart";
    case GC_EVENT_MARK_START:
        return "markstart";
    case GC_EVENT_MARK_END:
        return "markend";
    case GC_EVENT_RECLAIM_START:
        return "reclaimstart";
    case GC_EVENT_RECLAIM_END:
        return "reclaimend";
    case GC_EVENT_END: /* COLLECTION */
        return "clctend";
    case GC_EVENT_PRE_STOP_WORLD: /* STOPWORLD_BEGIN */
        return "prestopworld";
    case GC_EVENT_POST_STOP_WORLD: /* STOPWORLD_END */
        return "poststopworld";
    case GC_EVENT_PRE_START_WORLD: /* STARTWORLD_BEGIN */
        return "prestartworld";
    case GC_EVENT_POST_START_WORLD: /* STARTWORLD_END */
        return "poststartworld";
    case GC_EVENT_THREAD_SUSPENDED:
        return "threadsuspend";
    case GC_EVENT_THREAD_UNSUSPENDED:
        return "threadunsuspend";
    }
    assert(1==2);
}
#endif

