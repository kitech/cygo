// t18: crn_signal_interrupt_flag -> EINTR, clear -> normal sleep
#include "t00_common.h"

static crn_hchan* g_done;
static volatile int  g_rv;
static volatile int  g_errno;
static volatile long g_dt_ms;
static volatile long g_rem_ns;

static long t18_now_ms(void) {
    struct timespec t;
    clock_gettime(CLOCK_MONOTONIC, &t);
    return (long)t.tv_sec * 1000L + t.tv_nsec / 1000000L;
}

static void t18_sleeper(void* p) {
    (void)p;
    struct timespec req = { 5, 0 };
    struct timespec rem = { 0, 0 };
    long t0 = t18_now_ms();
    errno = 0;
    int rv = nanosleep(&req, &rem);
    long dt = t18_now_ms() - t0;

    g_rv = rv;
    g_errno = errno;
    g_dt_ms = dt;
    g_rem_ns = (long)rem.tv_sec * 1000000000L + rem.tv_nsec;
    linfo("T18 sleeper rv=%d errno=%d dt=%ldms rem=%ld.%09ld\n",
          rv, errno, dt, (long)rem.tv_sec, (long)rem.tv_nsec);
    t_join_send(g_done);
}

static void t18_signaler(void* p) {
    (void)p;
    usleep(1000 * 1000);   /* 1s */
    crn_signal_interrupt_flag = 1;
    linfo("T18 flag set %d\n", 1);
    t_join_send(g_done);
}

static void t18_controller(void* p) {
    (void)p;

    g_done = crn_hchan_new(0);
    g_rv = 0; g_errno = 0; g_dt_ms = 0; g_rem_ns = 0;
    crn_post(t18_sleeper, 0);
    crn_post(t18_signaler, 0);
    t_join_wait(g_done, 2);

    CHECK(g_rv == -1 && g_errno == EINTR,
          "signal: rv=%d errno=%d (want -1/EINTR)", g_rv, g_errno);
    CHECK(g_dt_ms < 2000, "signal: dt=%ldms (want < 2000)", g_dt_ms);
    CHECK(g_rem_ns > 0, "signal: rem=%ldns (want > 0)", g_rem_ns);

    crn_clear_signal_interrupt();

    g_rv = 0; g_errno = 0; g_dt_ms = 0; g_rem_ns = 0;
    crn_post(t18_sleeper, 0);
    t_join_wait(g_done, 1);

    CHECK(g_rv == 0, "after clear: rv=%d errno=%d (want 0)", g_rv, g_errno);
    CHECK(g_dt_ms >= 4900, "after clear: dt=%ldms (want >= 4900)", g_dt_ms);

    t_report_and_exit();
}

int main(void) {
    t_init();
    crn_post(t18_controller, 0);
    for (;;) sleep(1);
}
