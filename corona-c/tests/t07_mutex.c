// t07: crn_mutex contention — the unlock-wakeup deadlock fix regression
#include "t00_common.h"

#define NFIB 8
#define ROUNDS 1000
static crn_mutex* mu;
static long counter;
static crn_hchan* done;

static void locker(void* arg) {
    for (int i = 0; i < ROUNDS; i++) {
        crn_mutex_lock(mu);
        counter++; // non-atomic: protected only by the mutex
        if ((i % 100) == 0) { crn_sched(); } // stir the scheduler under load
        crn_mutex_unlock(mu);
        if ((i % 200) == 0) { linfo("locker fid=%d i=%d\n", crn_goid(), i); }
    }
    t_join_send(done);
}

static void controller(void* arg) {
    mu = crn_mutex_new();
    done = crn_hchan_new(0);
    for (int i = 0; i < NFIB; i++) { crn_post(locker, 0); }
    t_join_wait(done, NFIB);
    CHECK_EQ(counter, (long)NFIB * ROUNDS, "counter=%ld", counter);
    t_report_and_exit();
}

int main() {
    t_init();
    crn_post(controller, 0);
    for (;;) { sleep(1); }
}
