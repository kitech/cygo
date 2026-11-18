// t01: basic spawn/join + GC alloc inside fibers + thread stability
#include "t00_common.h"

#define NFIB 8
static crn_hchan* done;

static void worker(void* arg) {
    int id = (int)(uintptr_t)arg;
    int tid0 = gettid();
    for (int i = 0; i < 5; i++) {
        void* p = crn_gc_malloc(15550);
        CHECK(p != nilptr, "gc malloc %d/%d", id, i);
    }
    for (int i = 0; i < 3; i++) {
        sleep(0); // yield a bit
    }
    // thread must be the same across yields (main.c hello invariant)
    CHECK_EQ(gettid(), tid0, "tid changed worker %d", id);
    t_join_send(done);
}

static void controller(void* arg) {
    done = crn_hchan_new(0);
    CHECK(done != nilptr, "chan_new");
    for (int i = 0; i < NFIB; i++) {
        int id = crn_post(worker, (void*)(uintptr_t)(i + 1));
        CHECK(id > 0, "post id=%d", id);
    }
    GC_gcollect();
    t_join_wait(done, NFIB);
    t_report_and_exit();
}

int main() {
    t_init();
    crn_post(controller, 0);
    for (;;) { sleep(1); } // controller exits the process
}
