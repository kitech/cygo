#include <assert.h>
#include "norogc.h"

void* noro_raw_malloc(size_t size) {
    return calloc(1, size);
}
void* noro_raw_realloc(void* ptr, size_t size) {
    return realloc(ptr, size);
}
void noro_raw_free(void* ptr) {
    return free(ptr);
}

#ifdef USE_BDWGC

void* noro_gc_malloc(size_t size) {
    return GC_MALLOC(size);
}
void* noro_gc_realloc(void* ptr, size_t size) {
    return GC_REALLOC(ptr, size);
}
void noro_gc_free(void* ptr) {
    return GC_FREE(ptr);
}

const char* noro_gc_event_name(GC_EventType evty) {
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

