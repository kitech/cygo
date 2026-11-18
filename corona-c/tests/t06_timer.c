// t06: sleep/timer yield paths — timing bounds and concurrent sleepers
#include "t00_common.h"

static crn_hchan* done;

static long ms_since(struct timespec* t0) {
    struct timespec t1;
    clock_gettime(CLOCK_MONOTONIC, &t1);
    return (t1.tv_sec - t0->tv_sec) * 1000 + (t1.tv_nsec - t0->tv_nsec) / 1000000;
}

static void sleeper(void* arg) {
    struct timespec t0;
    clock_gettime(CLOCK_MONOTONIC, &t0);
    int which = (int)(uintptr_t)arg;
    if (which == 0) {
        usleep(50 * 1000); // 50ms
        long ms = ms_since(&t0);
        CHECK(ms >= 45 && ms <= 500, "usleep 50ms elapsed=%ld", ms);
    } else {
        struct timespec req = {0, 50 * 1000 * 1000};
        nanosleep(&req, nilptr);
        long ms = ms_since(&t0);
        CHECK(ms >= 45 && ms <= 500, "nanosleep 50ms elapsed=%ld", ms);
    }
    t_join_send(done);
}

static void quick_sleeper(void* arg) {
    usleep(5 * 1000); // 5ms
    t_join_send(done);
}

static void controller(void* arg) {
    done = crn_hchan_new(0);
    crn_post(sleeper, (void*)0);
    crn_post(sleeper, (void*)1);
    // concurrent sleepers: no double-resume (M13)
    for (int i = 0; i < 10; i++) { crn_post(quick_sleeper, 0); }
    t_join_wait(done, 12);
    t_report_and_exit();
}

int main() {
    t_init();
    crn_post(controller, 0);
    for (;;) { sleep(1); }
}
