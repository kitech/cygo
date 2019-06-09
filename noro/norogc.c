#include <assert.h>
#include "norogc.h"

void* noro_malloc(size_t size) {
    return NORO_MALLOC(size);
}
void noro_free(void* ptr){
    NORO_FREE(ptr);
}
void* noro_realloc(void* obj, size_t new_size){
    return NORO_REALLOC(obj, new_size);
}

#ifdef USE_BDWGC
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

