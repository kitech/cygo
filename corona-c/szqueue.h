#ifndef szqueue_h
#define szqueue_h


// Defines a circular buffer which acts as a FIFO queue.
typedef struct szqueue_t
{
    int    size;
    int    next;
    int    capacity;
    void** data;
} szqueue_t;

// Allocates and returns a new queue. The capacity specifies the maximum
// number of items that can be in the queue at one time. A capacity greater
// than INT_MAX / sizeof(void*) is considered an error. Returns NULL if
// initialization failed.
szqueue_t* szqueue_init(size_t capacity);

// Releases the queue resources.
void szqueue_dispose(szqueue_t* queue);

// Enqueues an item in the queue. Returns 0 if the add succeeded or -1 if it
// failed. If -1 is returned, errno will be set.
int szqueue_add(szqueue_t* queue, void* value);

// Dequeues an item from the head of the queue. Returns NULL if the queue is
// empty.
void* szqueue_remove(szqueue_t* queue);

// Returns, but does not remove, the head of the queue. Returns NULL if the
// queue is empty.
void* szqueue_peek(szqueue_t*);

#endif
