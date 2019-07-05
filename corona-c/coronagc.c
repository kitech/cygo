#include <assert.h>
#include <stdlib.h>
#include <stdio.h>

#include "coronagc.h"

/* void* crn_raw_malloc(size_t size) { */
/*     return calloc(1, size); */
/* } */
/* void* crn_raw_realloc(void* ptr, size_t size) { */
/*     return realloc(ptr, size); */
/* } */
/* void crn_raw_free(void* ptr) { */
/*     return free(ptr); */
/* } */

#ifdef USE_BDWGC

static void crn_gc_finalizer(void*ptr, void*clientdata) {
    printf("ptr dtor %p\n", ptr);
}
void* crn_gc_malloc(size_t size) {
    void* ptr = GC_MALLOC(size);
    // GC_register_finalizer(ptr, crn_gc_finalizer, 0, 0, 0);
    return ptr;
}
void* crn_gc_realloc(void* ptr, size_t size) {
    return GC_REALLOC(ptr, size);
}
void crn_gc_free(void* ptr) {
    GC_FREE(ptr);
}
void crn_gc_free2(void* ptr) {
    GC_FREE(ptr);
}
void* crn_gc_calloc(size_t n, size_t size) {
    return GC_MALLOC(n*size);
}

static void crn_finalizer_fwd(void* ptr, void (*ufin)(void* ptr)) { ufin(ptr);}
void crn_set_finalizer(void* ptr, void(*ufin)(void* ptr)) {
    GC_register_finalizer(ptr, crn_finalizer_fwd, ufin, 0, 0);
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

