
#include <cxrtbase.h>
#include <collectc/array.h>

struct array_s {
    size_t   size;
    size_t   capacity;
    float    exp_factor;
    void   **buffer;

    void *(*mem_alloc)  (size_t size);
    void *(*mem_calloc) (size_t blocks, size_t size);
    void  (*mem_free)   (void *block);

    size_t extra;
};
typedef struct array_s hkarray;

extern void* cxmalloc(size_t size);
extern void* cxcalloc(size_t blocks, size_t size);
extern void cxfree(void* ptr);

#define DEFAULT_CAPACITY 8
#define DEFAULT_EXPANSION_FACTOR 2
ArrayConf cxdftarconf = {.exp_factor = DEFAULT_EXPANSION_FACTOR,
                         .capacity   = DEFAULT_CAPACITY,
                         .mem_alloc = cxmalloc,
                         .mem_calloc = cxcalloc,
                         .mem_free = cxfree};

Array* cxarray_new() {
    Array* res = 0;
    ArrayConf arrconf = {0};
    arrconf.capacity = 1;
    arrconf.mem_alloc = cxmalloc;
    arrconf.mem_calloc = cxcalloc;
    arrconf.mem_free = cxfree;

    int rv = array_new_conf(&arrconf, &res);
    assert(rv == CC_OK);
    return res;
}

Array* cxarray_slice(Array* a0, int start, int end) {
    Array* narr = cxmalloc(sizeof(hkarray));
    memcpy(narr, a0, sizeof(Array));
    hkarray* hkarr = (hkarray*)narr;
    hkarr->size = end - start;
    hkarr->capacity = array_capacity(a0) - start;
    hkarr->buffer = ((hkarray*)a0)->buffer + start;

    return narr;
}

void* cxarray_get_at(Array* a0, int idx) {
    void* out = 0;
    int rv = array_get_at(a0, idx, &out);
    assert(rv == CC_OK);
    return out;
}

Array* cxarray_append(Array* a0, void* v) {
    int rv = array_add(a0, v);
    assert(rv == CC_OK);
    return a0;
}
