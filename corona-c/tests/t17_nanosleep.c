// t17: nanosleep early-wake diagnosis
#include "t00_common.h"

#ifndef T17_SLEEP_SEC
#define T17_SLEEP_SEC 3
#endif
#ifndef T17_SLEEPERS
#define T17_SLEEPERS 8
#endif
#ifndef T17_TICKERS
#define T17_TICKERS 8
#endif
#ifndef T17_ROUNDS
#define T17_ROUNDS 3
#endif

#define T17_MAX_SLEEPERS 32

typedef struct t17_arg { int idx; } t17_arg;

static crn_hchan* g_done;
static volatile long g_elapsed_ms[T17_MAX_SLEEPERS];
static volatile int  g_rv[T17_MAX_SLEEPERS];
static volatile int  g_errno[T17_MAX_SLEEPERS];

static int g_sleep_sec = T17_SLEEP_SEC;
static int g_sleepers  = T17_SLEEPERS;
static int g_tickers   = T17_TICKERS;
static int g_rounds    = T17_ROUNDS;

static long t17_now_ms(void) {
    struct timespec t;
    clock_gettime(CLOCK_MONOTONIC, &t);
    return (long)t.tv_sec * 1000L + t.tv_nsec / 1000000L;
}

static void t17_sleeper(void* p) {
    t17_arg* a = (t17_arg*)p;
    int i = a->idx;
    fiber* gr = crn_fiber_getcur();

    linfo("T17 sleeper[%d] start grid=%d mcid=%d state=%s wp=%d inprocer=%d\n",
          i, gr->id, gr->mcid, grstate2str(gr->state), gr->wake_pending,
          (int)crn_in_procer());

    struct timespec req = { g_sleep_sec, 0 };
    struct timespec rem = { 0, 0 };
    long t0 = t17_now_ms();
    errno = 0;
    int rv = nanosleep(&req, &rem);
    long dt = t17_now_ms() - t0;

    gr = crn_fiber_getcur();
    linfo("T17 sleeper[%d] end grid=%d rv=%d errno=%d dt=%ldms rem=%ld.%09ld state=%s wp=%d pkreason=%s\n",
          i, gr->id, rv, errno, dt, (long)rem.tv_sec, (long)rem.tv_nsec,
          grstate2str(gr->state), gr->wake_pending,
          yield_type_name(gr->pkreason));

    g_elapsed_ms[i] = dt;
    g_rv[i] = rv;
    g_errno[i] = errno;
    t_join_send(g_done);
}

static void t17_ticker(void* p) {
    (void)p;
    long end = t17_now_ms() + (g_sleep_sec + 1) * 1000L;
    int i = 0;
    while (t17_now_ms() < end) {
        usleep(1000 + (i++ % 5) * 1000);
    }
    t_join_send(g_done);
}

static void t17_controller(void* p) {
    (void)p;
    static t17_arg args[T17_MAX_SLEEPERS];
    int total_bad = 0;

    for (int r = 0; r < g_rounds; r++) {
        g_done = crn_hchan_new(0);
        for (int i = 0; i < g_sleepers; i++) {
            g_elapsed_ms[i] = -1;
            g_rv[i] = -999;
            g_errno[i] = -999;
        }
        for (int i = 0; i < g_sleepers; i++) {
            args[i].idx = i;
            crn_post(t17_sleeper, &args[i]);
        }
        for (int i = 0; i < g_tickers; i++) {
            crn_post(t17_ticker, 0);
        }
        t_join_wait(g_done, g_sleepers + g_tickers);

        int bad = 0;
        long expect = g_sleep_sec * 1000L;
        for (int i = 0; i < g_sleepers; i++) {
            if (g_elapsed_ms[i] < expect - 200) {
                bad++;
                printf("T17 EARLY round=%d sleeper=%d elapsed=%ldms rv=%d errno=%d\n",
                       r, i, g_elapsed_ms[i], g_rv[i], g_errno[i]);
            }
        }
        printf("T17 round=%d early=%d/%d\n", r, bad, g_sleepers);
        total_bad += bad;
    }

    CHECK(total_bad == 0, "nanosleep returned early %d time(s)", total_bad);
    t_report_and_exit();
}

static int t17_env_int(const char* name, int def) {
    const char* s = getenv(name);
    if (!s || !*s) return def;
    int v = atoi(s);
    return v < 0 ? def : v;
}

int main(void) {
    g_sleep_sec = t17_env_int("T17_SLEEP_SEC", T17_SLEEP_SEC);
    g_sleepers  = t17_env_int("T17_SLEEPERS",  T17_SLEEPERS);
    g_tickers   = t17_env_int("T17_TICKERS",   T17_TICKERS);
    g_rounds    = t17_env_int("T17_ROUNDS",    T17_ROUNDS);
    if (g_sleepers > T17_MAX_SLEEPERS) g_sleepers = T17_MAX_SLEEPERS;
    if (g_sleep_sec <= 0) g_sleep_sec = 1;
    if (g_rounds <= 0) g_rounds = 1;

    linfo("T17 config sec=%d sleepers=%d tickers=%d rounds=%d\n",
          g_sleep_sec, g_sleepers, g_tickers, g_rounds);

    t_init();
    crn_post(t17_controller, 0);
    for (;;) sleep(1);
}
