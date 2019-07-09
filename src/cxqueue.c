#include <collectc/queue.h>
#include <collectc/deque.h>

extern void* cxmalloc(size_t size);
extern void* cxcalloc(size_t blocks, size_t size);
extern void cxfree(void* ptr);

#define DEFAULT_CAPACITY 8
#define DEFAULT_EXPANSION_FACTOR 2

QueueConf cxdftfqconf = {.capacity   = DEFAULT_CAPACITY,
                        .mem_alloc  = cxmalloc,
                        .mem_calloc = cxcalloc,
                        .mem_free   = cxfree};

DequeConf cxdftdqconf = {.capacity   = DEFAULT_CAPACITY,
                         .mem_alloc  = cxmalloc,
                         .mem_calloc = cxcalloc,
                         .mem_free   = cxfree};
