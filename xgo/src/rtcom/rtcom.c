

#include <stdint.h>
#include <stdlib.h>

typedef struct {
    int (*incoro)();
    void* (*getcoro)();
    int (*yield)(long, int);
    int (*yield_multi)(int, int, long*, int*);

    // void (*fd_set_nonblocking)(int);
    // void (*fd_oncreate)(int, int, int, int, int, int);
    // void (*fd_onclose)(int);
    // void (*fd_ondup)(int, int);
} rtcom_yielder;

static rtcom_yielder rtcom_yielder_obj = {};

typedef struct {
    int (*resume_one)(void*grobj, int ytype, int grid, int mcid);
} rtcom_resumer;

static rtcom_resumer rtcom_resumer_obj = {};


typedef struct {
    void* (*mallocfn)(size_t);
    void* (*callocfn)(size_t, size_t);
    void* (*reallocfn)(void*, size_t);
    void (*freefn)(void*);
}rtcom_allocator;
static rtcom_allocator rtcom_allocer = {.mallocfn = malloc,
    .callocfn = calloc, .reallocfn = realloc, .freefn = free, };

#include <assert.h>
int rtcom_pre_gc_init(void* yielderx, uint32_t size0,
                      void* resumerx, uint32_t size1,
                      void* allocerx, uint32_t size2) {
    assert(size0 == sizeof(rtcom_yielder));
    assert(size1 == sizeof(rtcom_resumer));
    assert(size2 == sizeof(rtcom_allocator));
    assert(yielderx != 0);
    assert(resumerx != 0);

    rtcom_yielder_obj = *(rtcom_yielder*)yielderx;
    rtcom_resumer_obj = *(rtcom_resumer*)resumerx;
    if (allocerx != 0) {
        rtcom_allocer = *(rtcom_allocator*)allocerx;
    }

    return 0;
}

int rtcom_pre_gc_init2(void* incoro, void* getcoro, void* yield, void* yield_multi,
                       void* resumeone,
                       void* mallocfn, void* callocfn, void* reallocfn, void* freefn) {
    assert(incoro != NULL);
    assert(resumeone != NULL);
    //assert(mallocfn != NULL);

    rtcom_yielder_obj.incoro = incoro;
    rtcom_yielder_obj.getcoro = getcoro;
    rtcom_yielder_obj.yield = yield;
    rtcom_yielder_obj.yield_multi = yield_multi;

    rtcom_resumer_obj.resume_one = resumeone;
    if (mallocfn != NULL) {
        rtcom_allocer.mallocfn = mallocfn;
        rtcom_allocer.callocfn = callocfn;
        rtcom_allocer.reallocfn = reallocfn;
        rtcom_allocer.freefn = freefn;
    }

    //rtcom_yielder_obj = *(rtcom_yielder*)yielderx;
    //rtcom_resumer_obj = *(rtcom_resumer*)resumerx;
    //if (allocerx != 0) {
    //  rtcom_allocer = *(rtcom_allocator*)allocerx;
    //}

    return 0;
}

void* rtcom_yielder_get() { return &rtcom_yielder_obj; }
void* rtcom_resumer_get() { return &rtcom_resumer_obj; }
void* rtcom_allocator_get() { return &rtcom_allocer; }

