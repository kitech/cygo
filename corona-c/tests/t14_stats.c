// t14: goid uniqueness + crn_get_stats sanity while fibers run
#include "t00_common.h"

#define NFIB 20
static crn_hchan* done;

static void stats_user(void* arg) {
    // unique ids: ids must never repeat across live fibers (M7 fix)
    static _Atomic(int) seen[NFIB * 4];
    int id = crn_goid();
    CHECK(id > 0, "goid=%d", id);
    CHECK(id < NFIB * 4, "goid=%d out of test range", id);
    int prev = atomic_fetch_add(&seen[id], 1);
    CHECK_EQ(prev, 0, "goid=%d seen twice", id);

    crn_inner_stats st;
    memset(&st, 0, sizeof(st));
    crn_get_stats(&st);
    CHECK(st.fiber_totcnt > 0, "stats fiber_totcnt=%d", st.fiber_totcnt);
    CHECK(st.fiber_totmem > 0, "stats fiber_totmem=%d", st.fiber_totmem);
    // note: maxstksz depends on sampling time (used_stksz only updated when a
    // fiber suspends), so do not assert on it here
    t_join_send(done);
}

static void controller(void* arg) {
    done = crn_hchan_new(0);
    for (int i = 0; i < NFIB; i++) { crn_post(stats_user, 0); }
    t_join_wait(done, NFIB);
    t_report_and_exit();
}

int main() {
    t_init();
    crn_post(controller, 0);
    for (;;) { sleep(1); }
}
