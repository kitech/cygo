// t11: deep recursion + GC pressure inside fiber stacks — exercises the
// GC stack-scan path (crn_gc_push_other_roots1 / GC_push_all_eager)
#include "t00_common.h"

#define NFIB 6
#define DEPTH 64
static crn_hchan* done;

static long deep_sum(int d) {
    long pad[64]; // keep frames non-trivial so stacks really fill
    for (int i = 0; i < 64; i++) { pad[i] = d + i; }
    long s = 0;
    if (d >= DEPTH) {
        for (int i = 0; i < 64; i++) { s += pad[i]; }
        return s;
    }
    s = deep_sum(d + 1);
    for (int i = 0; i < 64; i++) { s += pad[i] * 2; }
    return s;
}

static void gc_recver(void* arg) {
    for (int i = 0; i < 30; i++) {
        deep_sum(0); // build deep stacks
        crn_gc_malloc(64 * 1024); // push GC to collect while stacks are deep
        if ((i % 7) == 0) { GC_gcollect(); }
    }
    t_join_send(done);
}

static void controller(void* arg) {
    done = crn_hchan_new(0);
    for (int i = 0; i < NFIB; i++) { crn_post(gc_recver, 0); }
    GC_gcollect();
    t_join_wait(done, NFIB);
    GC_gcollect();
    t_report_and_exit();
}

int main() {
    t_init();
    crn_post(controller, 0);
    for (;;) { sleep(1); }
}
