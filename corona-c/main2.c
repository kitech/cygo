// main2.c — long-running multi-scenario soak/chaos driver for corona.
//
// Unlike main.c (minimal demo), main2 exercises channels, select, close
// semantics, mutex/cond, timers, sockets, DNS, pipes/poll/select/epoll, file
// IO, GC pressure and fiber churn, and records per-scenario progress slots so
// a crash can be localized even when no stack is available (e.g. fiber stack
// overflow on macOS with no altstack handler).
//
// See MAIN2.md for the verification commands and scenario coverage.

#include "tests/t00_common.h"

#include <signal.h>

/* musl ships no <execinfo.h>; Alpine/Void provide it via libexecinfo.
 * Compile without it and say so, rather than faking a stack trace. */
#if __has_include(<execinfo.h>)
# include <execinfo.h>
# define CRN_HAVE_EXECINFO 1
#else
# define CRN_HAVE_EXECINFO 0
#endif
#include <errno.h>
#include <fcntl.h>
#include <poll.h>
#include <dirent.h>
#include <netinet/in.h>
#include <netinet/tcp.h>
#include <netdb.h>
#include <arpa/inet.h>
#include <sys/socket.h>
#include <sys/select.h>

#ifdef __linux__
# include <sys/epoll.h>
# include <sys/prctl.h>
# ifndef PR_SET_PTRACER
#  define PR_SET_PTRACER 0x59616d61
# endif
# ifndef PR_SET_PTRACER_ANY
#  define PR_SET_PTRACER_ANY ((unsigned long)-1)
# endif
# ifndef EFD_NONBLOCK
#  define EFD_NONBLOCK 0x800
# endif
extern int eventfd(unsigned int initval, int flags);
#endif

#include "iohook.h"

extern pid_t gettid();

// ------------------------------------------------------------------ options
static long g_deadline_ms;
static int g_seconds = 3600;
static int g_stats_sec = 30;
static int g_dns = 1;   /* DNS 默认开启；CRN2_DNS=0 可关闭 */
static int g_deep = 0;
static int g_overflow = 0;
static int g_abort_on_stall = 0;
static int g_destroy_on_exit = 0;
static unsigned g_seed = 1;
static volatile sig_atomic_t g_stopping = 0;
static volatile sig_atomic_t g_term_cnt = 0;
static volatile int g_controller_done = 0;
static volatile int g_controller_rc = 0;

// ------------------------------------------------------------- progress slots
enum {
    S1 = 0, S2, S3, S4, S5, S6, S7, S8, S9, S10,
    S11, S12, S13, S14, S15, S16, S17, S18, S19, S20,
    S21, S22, S23, S24, S25, S26, S27, S_MAX
};

static const char* g_snames[S_MAX] = {
    "s1_pingpong",   "s2_pipeline",   "s3_workerpool", "s4_selecttimeout",
    "s5_closestorm", "s6_mutex",      "s7_cond",       "s8_socketpair",
    "s9_tcp",        "s10_dns",       "s11_timer",     "s12_fiberchurn",
    "s13_gc",        "s14_sched",     "s15_pipepoll",  "s16_specifics",
    "s17_stackprobe","s18_overflow",  "s19_chanofchan","s20_semaphore",
    "s21_fairness",  "s22_closestate","s23_fdcheck",   "s24_epoll",
    "s25_select",    "s26_fileio",    "s27_exit",
};

static _Atomic int g_iter[S_MAX];
static _Atomic int g_op[S_MAX];
static _Atomic long g_hb[S_MAX];
static _Atomic int g_enabled[S_MAX];
static _Atomic int g_finished;
static _Atomic int g_posted;

static long now_ms(void) {
    struct timespec ts;
    clock_gettime(CLOCK_MONOTONIC, &ts);
    return ts.tv_sec * 1000 + ts.tv_nsec / 1000000;
}

static void tick(int s, int op) {
    atomic_store(&g_op[s], op);
    atomic_store(&g_hb[s], now_ms());
    atomic_fetch_add(&g_iter[s], 1);
}

// --------------------------------------------------------------- signal path
// async-signal-safe append: returns bytes written
static int app(char* dst, int cap, const char* s) {
    int n = 0;
    while (s[n] && n < cap) { dst[n] = s[n]; n++; }
    return n;
}
static int app_l(char* dst, int cap, long v) {
    char t[32]; int m = 0;
    if (v < 0) { if (m < cap) dst[m++] = '-'; v = -v; }
    if (v == 0) t[m++] = '0';
    while (v > 0 && m < 31) { t[m++] = '0' + (v % 10); v /= 10; }
    int n = 0;
    while (m > 0 && n < cap) { dst[n++] = t[--m]; }
    return n;
}
static int app_c(char* dst, int cap, char c) {
    if (cap <= 0) return 0;
    dst[0] = c;
    return 1;
}

static void on_fatal(int sig, siginfo_t* si, void* uc) {
    (void)uc;
    static char buf[4096];
    int n = 0;
    n += app(buf + n, sizeof(buf) - n, "\nSO-SIGNAL ");
    n += app_l(buf + n, sizeof(buf) - n, sig);
    n += app(buf + n, sizeof(buf) - n, " addr=");
    n += app_l(buf + n, sizeof(buf) - n, (long)(uintptr_t)(si ? si->si_addr : 0));
    n += app_c(buf + n, 1, '\n');
    long t = now_ms();
    for (int i = 0; i < S_MAX; i++) {
        if (!atomic_load(&g_enabled[i])) continue;
        n += app(buf + n, sizeof(buf) - n, "SO-SLOT ");
        n += app(buf + n, sizeof(buf) - n, g_snames[i]);
        n += app(buf + n, sizeof(buf) - n, " iter=");
        n += app_l(buf + n, sizeof(buf) - n, atomic_load(&g_iter[i]));
        n += app(buf + n, sizeof(buf) - n, " op=");
        n += app_l(buf + n, sizeof(buf) - n, atomic_load(&g_op[i]));
        n += app(buf + n, sizeof(buf) - n, " ago=");
        n += app_l(buf + n, sizeof(buf) - n, t - atomic_load(&g_hb[i]));
        n += app(buf + n, sizeof(buf) - n, "ms\n");
        if (n > (int)sizeof(buf) - 128) break;
    }
    write(2, buf, n);
#ifndef __APPLE__
# if CRN_HAVE_EXECINFO
    {
        void* bt[32];
        int m = backtrace(bt, 32);
        backtrace_symbols_fd(bt, m, 2);
    }
# else
    {
        /* signal handler: write() only (async-signal-safe) */
        static const char msg[] =
            "backtrace unavailable: this libc has no <execinfo.h> "
            "(musl: install libexecinfo / link -lexecinfo)\n";
        (void)!write(2, msg, sizeof(msg) - 1);
    }
# endif
#endif
    _exit(70);
}

// sigaltstack is per-thread state: every thread that can receive an
// SA_ONSTACK signal needs its OWN alternate stack. The old process-wide
// static buffer made all machine handlers share one 64KB region; two
// concurrent faults (e.g. a fiber guard-page hit plus a second machine
// during GC stop-world) stomp each other's frames - verified by probe
// (altprobe.c): shared -> immediate nested SIGSEGV, no diagnostics at all.
static void install_thread_altstack(void) {
    static __thread char altstack[64 * 1024];
    stack_t ss = {0};
    ss.ss_sp = altstack;
    ss.ss_size = sizeof(altstack);
    sigaltstack(&ss, 0);
}

static void install_fatal_handlers(void) {
    install_thread_altstack();

    struct sigaction sa;
    memset(&sa, 0, sizeof(sa));
    sa.sa_sigaction = on_fatal;
    sa.sa_flags = SA_SIGINFO | SA_ONSTACK;
    sigemptyset(&sa.sa_mask);
    sigaction(SIGSEGV, &sa, 0);
    sigaction(SIGBUS, &sa, 0);
    sigaction(SIGABRT, &sa, 0);
    sigaction(SIGILL, &sa, 0);
    sigaction(SIGFPE, &sa, 0);

    signal(SIGPIPE, SIG_IGN);
}

static void on_term(int sig) {
    (void)sig;
    g_stopping = 1;
    g_term_cnt++;
}
static void install_term_handlers(void) {
    struct sigaction sa;
    memset(&sa, 0, sizeof(sa));
    sa.sa_handler = on_term;
    sa.sa_flags = 0;
    sigemptyset(&sa.sa_mask);
    sigaction(SIGTERM, &sa, 0);
    sigaction(SIGINT, &sa, 0);
}

// main2.sh 超时后要 `gdb -p <pid>` 抓栈；gdb 是兄弟进程，Linux yama
// ptrace_scope=1 只允许祖先 attach，因此被测进程需自行放行。
// 仅在 CRN2_PTRACE=1 时生效，其余情况行为不变。
static void maybe_allow_ptrace(void) {
#ifdef __linux__
    const char* s = getenv("CRN2_PTRACE");
    if (s != NULL && s[0] == '1')
        prctl(PR_SET_PTRACER, PR_SET_PTRACER_ANY, 0, 0, 0);
#endif
}

// machine threads: altstack per thread (SIGSEGV handling on fiber overflow).
// sigaction is process-wide and already installed by main before machine
// creation; each thread only needs its own sigaltstack, not a re-install.
static void thread_create_cb(void* arg) {
    (void)arg;
    install_thread_altstack();
}

// ------------------------------------------------------------------- helpers
#define LOOP_WHILE(slot) \
    while (!g_stopping && now_ms() < g_deadline_ms)

// peers/scenarios report completion through a counter; the controller polls it
static void join_done(void) { atomic_fetch_add(&g_finished, 1); }
static void leave(int slot) { (void)slot; atomic_fetch_add(&g_finished, 1); }

static int set_nonblock(int fd) {
    int fl = fcntl(fd, F_GETFL, 0);
    if (fl < 0) return -1;
    return fcntl(fd, F_SETFL, fl | O_NONBLOCK);
}

// pipe2 is Linux-only; emulate with pipe + fcntl elsewhere (macOS/BSD)
static int mkpipe_nonblock(int p[2]) {
#ifdef __linux__
    return pipe2(p, O_NONBLOCK);
#else
    if (pipe(p) != 0) return -1;
    if (set_nonblock(p[0]) != 0 || set_nonblock(p[1]) != 0) {
        close(p[0]);
        close(p[1]);
        return -1;
    }
    return 0;
#endif
}

static int count_fds(void) {
#ifdef __linux__
    DIR* d = opendir("/proc/self/fd");
    if (!d) return -1;
    int n = 0;
    struct dirent* e;
    while ((e = readdir(d)) != NULL) {
        if (e->d_name[0] != '.') n++;
    }
    closedir(d);
    return n;
#else
    return -1;
#endif
}

// ------------------------------------------------------------------ s1 pingpong
static crn_hchan *s1a, *s1b;
static void s1_peer(void* arg) {
    (void)arg;
    while (!g_stopping) {
        void* v = nilptr;
        if (crn_hchan_recv(s1a, &v) != 1) break;
        if (crn_hchan_send(s1b, (void*)(uintptr_t)((uintptr_t)v + 1)) != 1) break;
    }
}
static void s1(void* arg) {
    (void)arg;
    s1a = crn_hchan_new(0);
    s1b = crn_hchan_new(0);
    crn_post(s1_peer, 0);
    uintptr_t i = 0;
    LOOP_WHILE(S1) {
        tick(S1, 0);
        if (crn_hchan_send(s1a, (void*)i) != 1) break;
        void* v = nilptr;
        tick(S1, 1);
        if (crn_hchan_recv(s1b, &v) != 1) break;
        if ((uintptr_t)v != i + 1) { CHECK(0, "s1 val %p != %lu", v, (unsigned long)(i + 1)); break; }
        i += 2;
    }
    crn_hchan_close(s1a);
    crn_hchan_close(s1b);
    leave(S1);
}

// ------------------------------------------------------------------ s2 pipeline
static void s2_stage(void* arg) {
    crn_hchan** ch = arg;
    while (!g_stopping) {
        void* v = nilptr;
        int rv = crn_hchan_recv(ch[0], &v);
        if (rv != 1) break;
        if (crn_hchan_send(ch[1], (void*)(uintptr_t)((uintptr_t)v * 2 + 1)) != 1) break;
    }
}
static void s2(void* arg) {
    (void)arg;
    LOOP_WHILE(S2) {
        crn_hchan* in = crn_hchan_new(4);
        crn_hchan* mid = crn_hchan_new(4);
        crn_hchan* out = crn_hchan_new(4);
        crn_hchan** ch1 = crn_gc_malloc(2 * sizeof(crn_hchan*));
        crn_hchan** ch2 = crn_gc_malloc(2 * sizeof(crn_hchan*));
        ch1[0] = in; ch1[1] = mid;
        ch2[0] = mid; ch2[1] = out;
        tick(S2, 0);
        crn_post(s2_stage, ch1);
        crn_post(s2_stage, ch2);
        for (int i = 0; i < 8; i++) {
            CHECK_EQ(crn_hchan_send(in, (void*)(uintptr_t)i), 1, "s2 send");
        }
        crn_hchan_close(in);
        for (int i = 0; i < 8; i++) {
            void* v = nilptr;
            tick(S2, 1);
            CHECK_EQ(crn_hchan_recv(out, &v), 1, "s2 recv");
            CHECK_EQ((uintptr_t)v, (uintptr_t)(i * 4 + 3), "s2 val %p i=%d", v, i);
        }
        crn_hchan_close(out);
        crn_hchan_close(mid);
        t_usleep_ms(1);
    }
    leave(S2);
}

// ------------------------------------------------------------------ s3 workerpool
static _Atomic int s3_handled;
static void s3_worker(void* arg) {
    crn_hchan** ch = arg;
    while (!g_stopping) {
        void* v = nilptr;
        if (crn_hchan_recv(ch[0], &v) != 1) break;
        atomic_fetch_add(&s3_handled, 1);
        if (crn_hchan_send(ch[1], v) != 1) break;
    }
}
static void s3(void* arg) {
    (void)arg;
    LOOP_WHILE(S3) {
        crn_hchan* jobs = crn_hchan_new(8);
        crn_hchan* res = crn_hchan_new(8);
        crn_hchan** ch = crn_gc_malloc(2 * sizeof(crn_hchan*));
        ch[0] = jobs; ch[1] = res;
        atomic_store(&s3_handled, 0);
        for (int w = 0; w < 4; w++) { tick(S3, 0); crn_post(s3_worker, ch); }
        for (int i = 0; i < 24; i++) { CHECK_EQ(crn_hchan_send(jobs, (void*)(uintptr_t)i), 1, "s3 send"); }
        crn_hchan_close(jobs);
        for (int i = 0; i < 24; i++) { tick(S3, 1); void* v = nilptr; CHECK_EQ(crn_hchan_recv(res, &v), 1, "s3 recv"); }
        CHECK_EQ(atomic_load(&s3_handled), 24, "s3 handled=%d", atomic_load(&s3_handled));
        crn_hchan_close(res);
    }
    leave(S3);
}

// ------------------------------------------------------------------ s4 select timeout
static void s4(void* arg) {
    (void)arg;
    crn_hchan* ch = crn_hchan_new(1);
    LOOP_WHILE(S4) {
        crn_scase* cs[2];
        cs[0] = crn_scase_new(ch, caseRecv, nilptr);
        cs[1] = crn_scase_new(ch, caseDefault, nilptr);
        int casi = -1;
        tick(S4, 0);
        bool ok = crn_goselect(&casi, cs, 2);
        crn_scase_free(cs[0]);
        crn_scase_free(cs[1]);
        if (ok) {
            // data path
        } else {
            CHECK_EQ(casi, 1, "s4 default casi=%d", casi);
            usleep(2 * 1000);
        }
        // occasionally feed it
        if ((atomic_load(&g_iter[S4]) % 7) == 0) {
            crn_hchan_send(ch, (void*)1);
        }
    }
    crn_hchan_close(ch);
    leave(S4);
}

// ------------------------------------------------------------------ s5 close storm
static void s5_parker_recv(void* arg) {
    crn_hchan* ch = arg;
    void* v = (void*)0xdeadbeef;
    int rv = crn_hchan_recv(ch, &v);
    CHECK_EQ(rv, 0, "s5 parked recv rv=%d", rv);
    CHECK(v == nilptr, "s5 parked recv val=%p", v);
    join_done();
}
static void s5_parker_send(void* arg) {
    crn_hchan* ch = arg;
    int rv = crn_hchan_send(ch, (void*)7);
    CHECK_EQ(rv, 0, "s5 parked send rv=%d", rv);
    join_done();
}
static void s5(void* arg) {
    (void)arg;
    LOOP_WHILE(S5) {
        // separate chans: a sender must never be consumed by our own recver,
        // otherwise the recv legitimately returns data (rv=1) instead of EOF
        crn_hchan* chR = crn_hchan_new(0);
        crn_hchan* chS = crn_hchan_new(0);
        int who = (int)(atomic_load(&g_iter[S5]) % 3);
        if (who == 0) {
            crn_post(s5_parker_recv, chR);
            crn_post(s5_parker_recv, chR);
        } else if (who == 1) {
            crn_post(s5_parker_send, chS);
        } else {
            crn_post(s5_parker_recv, chR);
            crn_post(s5_parker_send, chS);
        }
        tick(S5, 0);
        usleep(2 * 1000);
        CHECK_EQ(crn_hchan_close(chR), 1, "s5 closeR");
        CHECK_EQ(crn_hchan_close(chS), 1, "s5 closeS");
        CHECK_EQ(crn_hchan_close(chR), 1, "s5 double close");
        CHECK(crn_hchan_is_closed(chR), "s5 is_closed");
        void* v = (void*)0xdeadbeef;
        CHECK_EQ(crn_hchan_recv(chR, &v), 0, "s5 recv after close");
        CHECK_EQ(crn_hchan_send(chR, (void*)1), 0, "s5 send after close");
    }
    leave(S5);
}

// ------------------------------------------------------------------ s6 mutex
static crn_mutex* s6_mu;
static _Atomic long s6_counter;
static void s6_locker(void* arg) {
    while (!g_stopping && now_ms() < g_deadline_ms) {
        crn_mutex_lock(s6_mu);
        atomic_fetch_add(&s6_counter, 1);
        if ((atomic_load(&s6_counter) % 500) == 0) { crn_sched(); }
        crn_mutex_unlock(s6_mu);
        if ((atomic_load(&s6_counter) % 97) == 0) {
            if (crn_mutex_trylock(s6_mu) == 0) { crn_mutex_unlock(s6_mu); }
        }
    }
}
static void s6(void* arg) {
    (void)arg;
    s6_mu = crn_mutex_new();
    atomic_store(&s6_counter, 0);
    long start = atomic_load(&s6_counter);
    for (int i = 0; i < 4; i++) { tick(S6, 0); crn_post(s6_locker, 0); }
    while (!g_stopping && now_ms() < g_deadline_ms) {
        tick(S6, 1);
        usleep(20 * 1000);
    }
    long end = atomic_load(&s6_counter);
    CHECK(end >= start, "s6 counter regression %ld -> %ld", start, end);
    leave(S6);
}

// ------------------------------------------------------------------ s7 cond
static crn_mutex* s7_mu;
static crn_cond* s7_cd;
static _Atomic int s7_ready;
static void s7_waiter(void* arg) {
    (void)arg;
    while (!g_stopping && now_ms() < g_deadline_ms) {
        struct timespec ts;
        clock_gettime(CLOCK_REALTIME, &ts);
        ts.tv_nsec += 20 * 1000 * 1000;
        if (ts.tv_nsec >= 1000000000) { ts.tv_sec++; ts.tv_nsec -= 1000000000; }
        crn_mutex_lock(s7_mu);
        tick(S7, 0);
        int rv = crn_cond_timedwait(s7_cd, &ts);
        (void)rv;
        if (atomic_load(&s7_ready) > 0) atomic_fetch_sub(&s7_ready, 1);
        crn_mutex_unlock(s7_mu);
    }
    join_done();
}
static void s7(void* arg) {
    (void)arg;
    s7_mu = crn_mutex_new();
    s7_cd = crn_cond_new(s7_mu);
    crn_post(s7_waiter, 0);
    LOOP_WHILE(S7) {
        crn_mutex_lock(s7_mu);
        atomic_fetch_add(&s7_ready, 1);
        tick(S7, 1);
        crn_cond_signal(s7_cd);
        if ((atomic_load(&g_iter[S7]) % 5) == 0) crn_cond_broadcast(s7_cd);
        crn_mutex_unlock(s7_mu);
        usleep(5 * 1000);
    }
    leave(S7);
}

// ------------------------------------------------------------------ s8 socketpair
static void s8_reader(void* arg) {
    void** a = arg;
    int fd = (int)(uintptr_t)a[0];
    crn_hchan* done = a[1];
    char buf[32] = {0};
    int tries = 0;
    while (!g_stopping && tries < 5000) {
        int rv = recv(fd, buf, sizeof(buf) - 1, 0);
        if (rv > 0) { CHECK_EQ(rv, 5, "s8 recv rv=%d", rv); break; }
        tries++;
        usleep(1000);
    }
    crn_hchan_send(done, (void*)1);
    join_done();
}
static void s8(void* arg) {
    (void)arg;
    LOOP_WHILE(S8) {
        int sv[2];
        if (socketpair(AF_UNIX, SOCK_STREAM, 0, sv) != 0) { t_usleep_ms(1); continue; }
        set_nonblock(sv[0]);
        crn_hchan* done = crn_hchan_new(1);
        void** a = crn_gc_malloc(2 * sizeof(void*));
        a[0] = (void*)(uintptr_t)sv[0];
        a[1] = done;
        tick(S8, 0);
        crn_post(s8_reader, a);
        t_usleep_ms(1);
        CHECK_EQ(send(sv[1], "hello", 5, 0), 5, "s8 send");

        /* 等 reader 真正退出再 close，避免 close 与 recv 竞争。 */
        void* v = nilptr;
        CHECK_EQ(crn_hchan_recv(done, &v), 1, "s8 join");

        close(sv[0]);
        close(sv[1]);
        crn_hchan_close(done);
    }
    leave(S8);
}

// ------------------------------------------------------------------ s9 tcp echo
static void s9_client(void* arg) {
    int port = (int)(uintptr_t)arg;
    int fd = socket(AF_INET, SOCK_STREAM, 0);
    if (fd < 0) { join_done(); return; }
    struct sockaddr_in sa;
    memset(&sa, 0, sizeof(sa));
    sa.sin_family = AF_INET;
    sa.sin_port = htons((uint16_t)port);
    sa.sin_addr.s_addr = htonl(INADDR_LOOPBACK);
    set_nonblock(fd);
    int rv = connect(fd, (struct sockaddr*)&sa, sizeof(sa));
    if (rv != 0 && errno != EINPROGRESS) { close(fd); join_done(); return; }
    const char msg[] = "ping";
    for (int i = 0; i < 2000 && !g_stopping; i++) {
        rv = send(fd, msg, sizeof(msg) - 1, 0);
        if (rv == (int)(sizeof(msg) - 1)) break;
        usleep(1000);
    }
    char buf[16] = {0};
    for (int i = 0; i < 2000 && !g_stopping; i++) {
        rv = recv(fd, buf, sizeof(buf) - 1, 0);
        if (rv > 0) { CHECK_EQ(rv, 4, "s9 client recv=%d", rv); break; }
        usleep(1000);
    }
    close(fd);
    join_done();
}
static void s9(void* arg) {
    (void)arg;
    int ls = socket(AF_INET, SOCK_STREAM, 0);
    if (ls < 0) { linfo("s9 no socket %d\n", 0); leave(S9); return; }
    int on = 1;
    setsockopt(ls, SOL_SOCKET, SO_REUSEADDR, &on, sizeof(on));
    struct sockaddr_in sa;
    memset(&sa, 0, sizeof(sa));
    sa.sin_family = AF_INET;
    sa.sin_addr.s_addr = htonl(INADDR_LOOPBACK);
    sa.sin_port = 0;
    if (bind(ls, (struct sockaddr*)&sa, sizeof(sa)) != 0 || listen(ls, 8) != 0) {
        close(ls);
        leave(S9);
        return;
    }
    socklen_t sl = sizeof(sa);
    if (getsockname(ls, (struct sockaddr*)&sa, &sl) != 0) { close(ls); leave(S9); return; }
    int port = ntohs(sa.sin_port);
    set_nonblock(ls);
    LOOP_WHILE(S9) {
        tick(S9, 0);
        crn_post(s9_client, (void*)(uintptr_t)port);
        int cfd = accept(ls, 0, 0);
        if (cfd < 0) { usleep(2 * 1000); continue; }
        set_nonblock(cfd);
        char buf[32] = {0};
        int rv = recv(cfd, buf, sizeof(buf) - 1, 0);
        if (rv > 0) {
            tick(S9, 1);
            send(cfd, buf, rv, 0);
        }
        close(cfd);
        usleep(1 * 1000);
    }
    close(ls);
    leave(S9);
}

// ------------------------------------------------------------------ s10 dns
/*
 * DNS 默认执行（CRN2_DNS=0 可关闭）。
 * - 本地名（hosts / 数字地址）必须成功；
 * - 真实域名成功则做结构校验，失败只 WARN（离线/受限环境不判失败）；
 * - 真实域名限速（最多 1 秒 1 次），.invalid 最多 5 秒 1 次；
 * - 不校验具体 IP / 顺序 / ai_canonname：event 与 uv 后端这两项本来就不同。
 */
static const char* s10_hosts[] = {
    "localhost",
    "127.0.0.1",
    "::1",
    "example.com",
    "www.example.org",
    "one.one.one.one",
    "crn-no-such-host.invalid",
};

static int s10_is_local(const char* h) {
    return strcmp(h, "localhost") == 0 ||
           strcmp(h, "127.0.0.1") == 0 ||
           strcmp(h, "::1") == 0;
}

static int s10_expect_fail(const char* h) {
    return strcmp(h, "crn-no-such-host.invalid") == 0;
}

/* 直接走 hooked getaddrinfo()，校验后端返回的 addrinfo 结构与所有权。
 * 返回条目数；-1 表示可容忍的解析失败（真实域名/预期失败域名）。 */
static int s10_check_addrinfo(const char* host) {
    struct addrinfo h;
    memset(&h, 0, sizeof(h));
    h.ai_family = AF_UNSPEC;
    h.ai_socktype = SOCK_STREAM;
    h.ai_flags = AI_CANONNAME;

    struct addrinfo* res = nilptr;
    int rv = getaddrinfo(host, NULL, &h, &res);

    if (rv != 0) {
        CHECK(res == nilptr, "s10 %s rv=%d res!=nil", host, rv);
        if (s10_is_local(host)) {
            CHECK(0, "s10 local %s rv=%d", host, rv);
        } else if (!s10_expect_fail(host)) {
            lwarn("s10 %s rv=%d（外部域名，离线/受限时容忍）\n", host, rv);
        }
        return -1;
    }

    CHECK(res != nilptr, "s10 %s rv=0 res=nil", host);
    int n = 0;
    for (struct addrinfo* ai = res; ai != nilptr; ai = ai->ai_next) {
        CHECK(ai->ai_family == AF_INET || ai->ai_family == AF_INET6,
              "s10 %s family=%d", host, ai->ai_family);
        CHECK(ai->ai_addr != nilptr, "s10 %s addr=nil", host);
        size_t need = ai->ai_family == AF_INET ? sizeof(struct sockaddr_in)
                                               : sizeof(struct sockaddr_in6);
        CHECK((size_t)ai->ai_addrlen >= need,
              "s10 %s addrlen=%u need=%lu", host,
              (unsigned)ai->ai_addrlen, (unsigned long)need);
        if (ai->ai_addr != nilptr) {
            char abuf[INET6_ADDRSTRLEN] = {0};
            void* p = ai->ai_family == AF_INET
                    ? (void*)&((struct sockaddr_in*)ai->ai_addr)->sin_addr
                    : (void*)&((struct sockaddr_in6*)ai->ai_addr)->sin6_addr;
            CHECK(inet_ntop(ai->ai_family, p, abuf, sizeof(abuf)) != nilptr &&
                  abuf[0] != '\0',
                  "s10 %s inet_ntop family=%d", host, ai->ai_family);
        }
        n++;
    }
    CHECK(n >= 1, "s10 %s no entries", host);
    if (s10_expect_fail(host)) {
        lwarn("s10 %s 被解析为有效地址（DNS 劫持？仅记录）\n", host);
    }
    freeaddrinfo(res);
    return n;
}

static void s10(void* arg) {
    (void)arg;
    extern char* crn_getaddrinfo(const char* hostname);
    const int nhost = (int)(sizeof(s10_hosts) / sizeof(s10_hosts[0]));
    int idx = 0;
    long last_real = 0;
    long last_bad = 0;

    LOOP_WHILE(S10) {
        const char* host = s10_hosts[idx++ % nhost];
        int is_local = s10_is_local(host);
        int is_bad = s10_expect_fail(host);
        long now = now_ms();

        /* 限速：外部域名 1 秒最多 1 次，预期失败域名 5 秒最多 1 次。 */
        if (!is_local && !is_bad && now - last_real < 1000) {
            tick(S10, 0);
            usleep(20 * 1000);
            continue;
        }
        if (is_bad && now - last_bad < 5000) {
            tick(S10, 0);
            usleep(20 * 1000);
            continue;
        }
        if (!is_local && !is_bad) last_real = now;
        if (is_bad) last_bad = now;

        tick(S10, 0);
        char* ip = crn_getaddrinfo(host);
        if (is_local) {
            CHECK(ip != nilptr, "s10 crn_getaddrinfo(%s)=nil", host);
        } else if (ip == nilptr) {
            if (!is_bad) {
                lwarn("s10 crn_getaddrinfo(%s)=nil（外部域名，容忍）\n", host);
            }
        }
        if (ip != nilptr) {
            struct in_addr a4;
            struct in6_addr a6;
            CHECK(ip[0] != '\0', "s10 %s empty ip", host);
            CHECK(inet_pton(AF_INET, ip, &a4) == 1 ||
                  inet_pton(AF_INET6, ip, &a6) == 1,
                  "s10 %s bad ip=%s", host, ip);
            if (is_bad) {
                lwarn("s10 crn_getaddrinfo(%s) 返回有效地址（DNS 劫持？仅记录）\n", host);
            }
        }

        tick(S10, 1);
        (void)s10_check_addrinfo(host);

#ifdef __linux__
        if (strcmp(host, "localhost") == 0) {
            struct hostent he;
            struct hostent* hr = nilptr;
            char hbuf[1024] = {0};
            int herr = 0;
            tick(S10, 2);
            int rv = gethostbyname_r(host, &he, hbuf, sizeof(hbuf) - 1, &hr, &herr);
            CHECK(rv == 0 && hr != nilptr,
                  "s10 gethostbyname_r(%s) rv=%d herr=%d", host, rv, herr);
        }
#endif
        usleep(20 * 1000);
    }
    leave(S10);
}

// ------------------------------------------------------------------ s11 timers
static void s11_sleeper(void* arg) {
    unsigned seed = (unsigned)(uintptr_t)arg ^ g_seed;
    while (!g_stopping && now_ms() < g_deadline_ms) {
        int us = 1 + (int)(rand_r(&seed) % 8000);
        if (us & 1) usleep(us); else {
            struct timespec rq = {0, (long)us * 1000};
            nanosleep(&rq, 0);
        }
    }
    join_done();
}
static void s11(void* arg) {
    (void)arg;
    for (int i = 0; i < 16; i++) { tick(S11, 0); crn_post(s11_sleeper, (void*)(uintptr_t)(i + 1)); }
    while (!g_stopping && now_ms() < g_deadline_ms) {
        tick(S11, 1);
        usleep(50 * 1000);
    }
    leave(S11);
}

// ------------------------------------------------------------------ s12 fiber churn
static void s12_short(void* arg) {
    crn_hchan* done = arg;
    void* p = crn_gc_malloc(64);
    (void)p;
    crn_hchan_send(done, (void*)1);
    join_done();
}
static void s12(void* arg) {
    (void)arg;
    LOOP_WHILE(S12) {
        crn_hchan* done = crn_hchan_new(32);
        for (int i = 0; i < 32; i++) {
            tick(S12, 0);
            int rv = crn_post(s12_short, done);
            CHECK(rv > 0, "s12 post rv=%d", rv);
        }
        /* 真正等待这一批完成；若 1 号机派发不出去，s12 会阻塞在这里，
         * 不会继续 crn_post，从而把待派发 fiber 限制在一批 32 个。 */
        for (int i = 0; i < 32; i++) {
            tick(S12, 1);
            void* v = nilptr;
            int rv = crn_hchan_recv(done, &v);
            CHECK(rv == 1, "s12 join rv=%d", rv);
        }
        crn_hchan_close(done);

        crn_inner_stats st;
        memset(&st, 0, sizeof(st));
        crn_get_stats(&st);
        CHECK(st.fiber_totcnt > 0, "s12 stats fibers=%d", st.fiber_totcnt);

        crn_fiber_info fi[64];
        int nf = 0;
        for (int t = 0; t < 10 && nf <= 0; t++) {
            nf = crn_get_fibers(fi, (int)(sizeof(fi) / sizeof(fi[0])));
            if (nf <= 0) usleep(1000);
        }
        CHECK(nf > 0, "s12 fiber list nf=%d", nf);
        int ncheck = nf < (int)(sizeof(fi) / sizeof(fi[0]))
                   ? nf : (int)(sizeof(fi) / sizeof(fi[0]));
        for (int i = 0; i < ncheck; i++) {
            CHECK(fi[i].id > 0, "s12 fiber[%d] id=%d", i, fi[i].id);
            CHECK(fi[i].mcid != 0 && fi[i].mcid != 2,
                  "s12 fiber[%d] mcid=%d", i, fi[i].mcid);
            CHECK(fi[i].state >= 0 && fi[i].state <= 4,
                  "s12 fiber[%d] state=%d", i, fi[i].state);
            CHECK(fi[i].stksz > 0, "s12 fiber[%d] stksz=%zu", i, fi[i].stksz);
        }
        CHECK(crn_num_fibers() > 0, "s12 crn_num_fibers=0");
    }
    leave(S12);
}

// ------------------------------------------------------------------ s13 GC
static _Atomic int s13_fin;
static void s13_fin_cb(void* p) {
    (void)p;
    atomic_fetch_add(&s13_fin, 1);
}
static void s13(void* arg) {
    (void)arg;
    LOOP_WHILE(S13) {
        for (int i = 0; i < 64; i++) {
            size_t sz = (i % 8 == 0) ? 64 * 1024 : 128;
            void* p = (i % 3 == 0) ? crn_gc_calloc(1, sz) : crn_gc_malloc(sz);
            CHECK(p != nilptr, "s13 alloc");
            if ((i % 5) == 0) {
                crn_set_finalizer(p, s13_fin_cb);
            }
            if ((i % 7) == 0) {
                void* q = crn_gc_realloc(p, sz + 32);
                CHECK(q != nilptr, "s13 realloc");
            }
            if ((i % 11) == 0) {
                void* u = crn_gc_malloc_uncollectable(256);
                crn_gc_free_uncollectable(u);
            }
        }
        tick(S13, 0);
        if ((atomic_load(&g_iter[S13]) % 20) == 0) GC_gcollect();
        usleep(2 * 1000);
    }
    linfo("s13 finalizers=%d %d\n", atomic_load(&s13_fin), 0);
    leave(S13);
}

// ------------------------------------------------------------------ s14 sched
static void s14(void* arg) {
    (void)arg;
    LOOP_WHILE(S14) {
        tick(S14, 0);
        for (int i = 0; i < 200; i++) crn_sched();
        usleep(1 * 1000);
    }
    leave(S14);
}

// ------------------------------------------------------------------ s15 pipe/poll
static void s15(void* arg) {
    (void)arg;
    int pfd[2];
    if (mkpipe_nonblock(pfd) != 0) { leave(S15); return; }
#ifdef __linux__
    int efd = eventfd(0, EFD_NONBLOCK);
#else
    int efd = -1; // eventfd is Linux-only
#endif
    LOOP_WHILE(S15) {
        char c = 'x';
        tick(S15, 0);
        if (write(pfd[1], &c, 1) == 1) {
            struct pollfd pf = {.fd = pfd[0], .events = POLLIN};
            int rv = poll(&pf, 1, 10);
            if (rv > 0) { char o = 0; CHECK_EQ(read(pfd[0], &o, 1), 1, "s15 read"); }
        }
        if (efd >= 0) {
            uint64_t v = 1;
            tick(S15, 1);
            write(efd, &v, sizeof(v));
            uint64_t ov = 0;
            read(efd, &ov, sizeof(ov));
            struct pollfd pf2 = {.fd = efd, .events = POLLIN};
            poll(&pf2, 1, 1);
        }
    }
    close(pfd[0]); close(pfd[1]);
    if (efd >= 0) close(efd);
    leave(S15);
}

// ------------------------------------------------------------------ s16 specifics
static void s16(void* arg) {
    (void)arg;
    static __thread int key;
    void* mine = (void*)(uintptr_t)(0x1000 + (uintptr_t)arg);
    crn_fiber_setspec(&key, mine);
    LOOP_WHILE(S16) {
        tick(S16, 0);
        void* v = crn_fiber_getspec(&key);
        CHECK_EQ((uintptr_t)v, (uintptr_t)mine, "s16 spec %p != %p", v, mine);
        usleep(2 * 1000);
    }
    leave(S16);
}

// ------------------------------------------------------------------ s17 stack probe
static unsigned long s17_recurse(int d) {
    volatile unsigned char pad[512];
    for (int i = 0; i < (int)sizeof(pad); i += 64) pad[i] = (unsigned char)(d + i);
    if (d <= 0) return (unsigned long)pad[0];
    return s17_recurse(d - 1) + pad[0];
}
static void s17(void* arg) {
    (void)arg;
    void* addr = nilptr;
    size_t size = 0;
    int rv = crn_fiber_stackaddr_cur(&addr, &size);
    linfo("s17 stack addr=%p size=%lu %d\n", addr, (unsigned long)size, rv);
    int depth = g_deep ? 400 : 40;
    LOOP_WHILE(S17) {
        tick(S17, 0);
        unsigned long v = s17_recurse(depth);
        (void)v;
        usleep(5 * 1000);
    }
    leave(S17);
}

// ------------------------------------------------------------------ s18 overflow
static unsigned long s18_recurse(int d) {
    volatile unsigned char pad[1024];
    pad[0] = (unsigned char)d;
    return s18_recurse(d + 1) + pad[0];
}
static void s18(void* arg) {
    (void)arg;
    tick(S18, 0);
    linfo("s18 deliberate stack overflow %d\n", 0);
    s18_recurse(0); // never returns
    leave(S18);
}

// ------------------------------------------------------------------ s19 chan of chans
static void s19_receiver(void* arg) {
    crn_hchan* outer = arg;
    while (!g_stopping) {
        void* v = nilptr;
        if (crn_hchan_recv(outer, &v) != 1) break;
        crn_hchan* inner = v;
        void* got = nilptr;
        if (crn_hchan_recv(inner, &got) != 1) break;
        if ((uintptr_t)got != 42) { CHECK(0, "s19 got %p", got); break; }
        crn_hchan_close(inner);
    }
    join_done();
}
static void s19(void* arg) {
    (void)arg;
    crn_hchan* outer = crn_hchan_new(2);
    crn_post(s19_receiver, outer);
    LOOP_WHILE(S19) {
        crn_hchan* inner = crn_hchan_new(1);
        tick(S19, 0);
        CHECK_EQ(crn_hchan_send(inner, (void*)42), 1, "s19 inner send");
        CHECK_EQ(crn_hchan_send(outer, inner), 1, "s19 outer send");
        usleep(2 * 1000);
    }
    crn_hchan_close(outer);
    leave(S19);
}

// ------------------------------------------------------------------ s20 semaphore
static crn_hchan* s20_sem;
static _Atomic int s20_active;
static _Atomic int s20_max;
static void s20_worker(void* arg) {
    (void)arg;
    while (!g_stopping && now_ms() < g_deadline_ms) {
        void* tok = nilptr;
        if (crn_hchan_recv(s20_sem, &tok) != 1) break;
        int cur = atomic_fetch_add(&s20_active, 1) + 1;
        int prev = atomic_load(&s20_max);
        while (cur > prev && !atomic_compare_exchange_weak(&s20_max, &prev, cur)) {}
        crn_sched();
        atomic_fetch_sub(&s20_active, 1);
        crn_hchan_send(s20_sem, (void*)1);
    }
    join_done();
}
static void s20(void* arg) {
    (void)arg;
    s20_sem = crn_hchan_new(4);
    for (int i = 0; i < 4; i++) crn_hchan_send(s20_sem, (void*)1);
    for (int i = 0; i < 8; i++) { tick(S20, 0); crn_post(s20_worker, 0); }
    while (!g_stopping && now_ms() < g_deadline_ms) {
        tick(S20, 1);
        usleep(20 * 1000);
    }
    CHECK(atomic_load(&s20_max) <= 4, "s20 max concurrency=%d", atomic_load(&s20_max));
    leave(S20);
}

// ------------------------------------------------------------------ s21 fairness
static void s21_feeder(void* arg) {
    crn_hchan** ch = arg;
    unsigned s = g_seed;
    while (!g_stopping) {
        int which = (int)(rand_r(&s) % 2);
        if (crn_hchan_send(ch[which], (void*)(uintptr_t)(which + 1)) != 1) break;
    }
    join_done();
}
static void s21(void* arg) {
    (void)arg;
    crn_hchan* c1 = crn_hchan_new(0);
    crn_hchan* c2 = crn_hchan_new(0);
    static crn_hchan* ch[2];
    ch[0] = c1; ch[1] = c2;
    crn_post(s21_feeder, ch);
    int cnt[2] = {0, 0};
    int n = 0;
    LOOP_WHILE(S21) {
        crn_scase* cs[2];
        cs[0] = crn_scase_new(c1, caseRecv, nilptr);
        cs[1] = crn_scase_new(c2, caseRecv, nilptr);
        int casi = -1;
        tick(S21, 0);
        bool ok = crn_goselect(&casi, cs, 2);
        crn_scase_free(cs[0]);
        crn_scase_free(cs[1]);
        if (ok && (casi == 0 || casi == 1)) cnt[casi]++;
        if (++n >= 200) {
            int d = cnt[0] - cnt[1];
            if (d < 0) d = -d;
            CHECK(d <= 120, "s21 unfair %d vs %d", cnt[0], cnt[1]);
            cnt[0] = cnt[1] = 0;
            n = 0;
        }
    }
    crn_hchan_close(c1);
    crn_hchan_close(c2);
    leave(S21);
}

// ------------------------------------------------------------------ s22 close state machine
static void s22(void* arg) {
    (void)arg;
    LOOP_WHILE(S22) {
        crn_hchan* ch = crn_hchan_new(4);
        tick(S22, 0);
        CHECK_EQ(crn_hchan_cap(ch), 4, "s22 cap");
        CHECK_EQ(crn_hchan_send(ch, (void*)1), 1, "s22 send");
        CHECK_EQ(crn_hchan_len(ch), 1, "s22 len before close");
        CHECK_EQ(crn_hchan_close(ch), 1, "s22 close");
        CHECK_EQ(crn_hchan_close(ch), 1, "s22 close2");
        CHECK(crn_hchan_is_closed(ch), "s22 closed");
        void* v = (void*)0xdeadbeef;
        CHECK_EQ(crn_hchan_recv(ch, &v), 0, "s22 recv closed rv");
        CHECK_EQ(crn_hchan_send(ch, (void*)2), 0, "s22 send closed");
        tick(S22, 1);
        usleep(1 * 1000);
    }
    leave(S22);
}

// ------------------------------------------------------------------ s23 fd check
static void s23(void* arg) {
    (void)arg;
    int base = count_fds();
    if (base < 0) { leave(S23); return; }
    int n = 0;
    LOOP_WHILE(S23) {
        int sv[2];
        if (socketpair(AF_UNIX, SOCK_STREAM, 0, sv) == 0) { close(sv[0]); close(sv[1]); }
        int p[2];
        if (pipe(p) == 0) { close(p[0]); close(p[1]); }
        tick(S23, 0);
        if (++n >= 200) {
            int cur = count_fds();
            CHECK(cur - base <= 16, "s23 fd growth base=%d cur=%d", base, cur);
            n = 0;
            usleep(5 * 1000);
        }
    }
    leave(S23);
}

// ------------------------------------------------------------------ s24 epoll
static void s24(void* arg) {
    (void)arg;
#ifndef __linux__
    // epoll is Linux-only; this scenario is disabled elsewhere
    leave(S24);
    return;
#else
    int ep = epoll_create1(EPOLL_CLOEXEC);
    int efd = eventfd(0, EFD_NONBLOCK);
    if (ep < 0 || efd < 0) {
        if (ep >= 0) close(ep);
        if (efd >= 0) close(efd);
        return;
    }
    struct epoll_event ev = {0};
    ev.events = EPOLLIN;
    ev.data.fd = efd;
    epoll_ctl(ep, EPOLL_CTL_ADD, efd, &ev);
    LOOP_WHILE(S24) {
        uint64_t v = 1;
        write(efd, &v, sizeof(v));
        struct epoll_event out = {0};
        tick(S24, 0);
        epoll_wait(ep, &out, 1, 10);
        uint64_t ov = 0;
        read(efd, &ov, sizeof(ov));
        usleep(1 * 1000);
    }
    close(ep);
    close(efd);
    leave(S24);
#endif
}

// ------------------------------------------------------------------ s25 select
static void s25(void* arg) {
    (void)arg;
    int p[2];
    if (mkpipe_nonblock(p) != 0) { leave(S25); return; }
    LOOP_WHILE(S25) {
        char c = 'y';
        write(p[1], &c, 1);
        fd_set rf;
        FD_ZERO(&rf);
        FD_SET(p[0], &rf);
        struct timeval tv = {0, 10 * 1000};
        tick(S25, 0);
        int rv = select(p[0] + 1, &rf, 0, 0, &tv);
        if (rv > 0) { char o; read(p[0], &o, 1); }
    }
    close(p[0]);
    close(p[1]);
    leave(S25);
}

// ------------------------------------------------------------------ s26 file io
static void s26(void* arg) {
    (void)arg;
    char path[128];
    const char* tmp = getenv("CRN2_TMPDIR");
    if (tmp == NULL || tmp[0] == '\0') tmp = getenv("TMPDIR");
    if (tmp == NULL || tmp[0] == '\0') tmp = "/tmp";
    snprintf(path, sizeof(path), "%s/main2_io.%d", tmp, (int)getpid());
    LOOP_WHILE(S26) {
        int fd = open(path, O_CREAT | O_RDWR | O_TRUNC, 0600);
        if (fd >= 0) {
            tick(S26, 0);
            char b[64] = "corona-main2";
            write(fd, b, sizeof(b));
            lseek(fd, 0, SEEK_SET);
            char o[64] = {0};
            read(fd, o, sizeof(o));
            close(fd);
            unlink(path);
        }
        FILE* f = fopen(path, "w");
        if (f) {
            tick(S26, 1);
            fputs("x", f);
            fclose(f);
            unlink(path);
        }
        usleep(3 * 1000);
    }
    leave(S26);
}

// ------------------------------------------------------------------- runoff
static void* run_scenario(int id) {
    switch (id) {
    case S1: return s1;   case S2: return s2;   case S3: return s3;
    case S4: return s4;   case S5: return s5;   case S6: return s6;
    case S7: return s7;   case S8: return s8;   case S9: return s9;
    case S10: return s10; case S11: return s11; case S12: return s12;
    case S13: return s13; case S14: return s14; case S15: return s15;
    case S16: return s16; case S17: return s17; case S18: return s18;
    case S19: return s19; case S20: return s20;
    case S21: return s21; case S22: return s22; case S23: return s23;
    case S24: return s24; case S25: return s25; case S26: return s26;
    default: return 0;
    }
}

static void watchdog(void* arg) {
    (void)arg;
    linfo("watchdog: enter tid=%d\n", gettid());
    while (!g_stopping && now_ms() < g_deadline_ms) {
        sleep(5);
        long t = now_ms();
        for (int i = 0; i < S_MAX; i++) {
            if (!atomic_load(&g_enabled[i])) continue;
            long ago = t - atomic_load(&g_hb[i]);
            if (ago > 120 * 1000) {
                lwarn("STALL %s iter=%d op=%d ago=%ldms\n",
                      g_snames[i], atomic_load(&g_iter[i]), atomic_load(&g_op[i]), ago);
                if (g_abort_on_stall) abort();
            }
        }
    }
    linfo("watchdog: exit stopping=%d now=%ld\n", (int)g_stopping, now_ms());
}

static void stats_dump(void) {
    crn_inner_stats st;
    memset(&st, 0, sizeof(st));
    crn_get_stats(&st);
    linfo("STATS t=%ld fibers=%d act=%d mem=%d maxstk=%d mch=%d/%d fin=%d\n",
          now_ms() / 1000, st.fiber_totcnt, st.fiber_actcnt, st.fiber_totmem,
          st.maxstksz, st.mch_actcnt, st.mch_totcnt, atomic_load(&g_finished));
}

static void controller(void* arg) {
    (void)arg;
    linfo("controller: enter tid=%d\n", gettid());
    for (int i = 0; i < S_MAX; i++) {
        if (!atomic_load(&g_enabled[i])) continue;
        void (*entry)(void*) = run_scenario(i);
        if (!entry) continue;
        linfo("controller: post %s\n", g_snames[i]);
        int rv = crn_post(entry, 0);
        CHECK(rv > 0, "post %s rv=%d", g_snames[i], rv);
        atomic_fetch_add(&g_posted, 1);
    }
    linfo("posted %d scenarios %d\n", atomic_load(&g_posted), 0);
    crn_post(watchdog, 0);
    linfo("controller: watchdog posted tid=%d\n", gettid());

    long stats_at = now_ms() + g_stats_sec * 1000L;
    while (!g_stopping && now_ms() < g_deadline_ms) {
        sleep(1);
        if (now_ms() >= stats_at) { stats_dump(); stats_at = now_ms() + g_stats_sec * 1000L; }
    }
    linfo("controller: loop exit stopping=%d sigterm=%d now=%ld deadline=%ld\n",
          (int)g_stopping, (int)g_term_cnt, now_ms(), g_deadline_ms);
    g_stopping = 1;
    linfo("controller: drain wait posted=%d finished=%d\n",
          atomic_load(&g_posted), atomic_load(&g_finished));
    // let scenarios unwind
    for (int i = 0; i < 30; i++) {
        if (atomic_load(&g_finished) >= atomic_load(&g_posted)) break;
        usleep(100 * 1000);
    }
    stats_dump();
    int posted = atomic_load(&g_posted);
    int fin = atomic_load(&g_finished);
    linfo("controller: drain done %d/%d\n", fin, posted);
    linfo("DONE posted=%d finished=%d\n", posted, fin);
    if (t_fails > 0) { linfo("FAIL checks=%d\n", t_fails); }
    if (g_destroy_on_exit) {
        linfo("controller: crn_destroy begin tid=%d\n", gettid());
        crn_destroy(crn_get());
        linfo("controller: crn_destroy done tid=%d\n", gettid());
    }
    g_controller_rc = (t_fails > 0 || fin < posted) ? 1 : 0;
    linfo("controller: set done rc=%d\n", g_controller_rc);
    g_controller_done = 1;
}

// ------------------------------------------------------------------- init/main
static unsigned env_u(const char* name, unsigned def) {
    const char* s = getenv(name);
    if (!s || !*s) return def;
    return (unsigned)strtoul(s, 0, 0);
}
static int env_i(const char* name, int def) {
    const char* s = getenv(name);
    if (!s || !*s) return def;
    return atoi(s);
}

static int scenario_selected(const char* name) {
    const char* sel = getenv("CRN2_SCENARIOS");
    if (!sel || !*sel) return 1;
    // match either "s5" or full name
    const char* p = sel;
    while (*p) {
        const char* e = strchr(p, ',');
        size_t n = e ? (size_t)(e - p) : strlen(p);
        if (n > 0) {
            if (n == strlen(name) && strncmp(p, name, n) == 0) return 1;
            char num[8] = {0};
            snprintf(num, sizeof(num), "s%d", atoi(name + 1));
            if (strlen(num) == n && strncmp(p, num, n) == 0) return 1;
        }
        if (!e) break;
        p = e + 1;
    }
    return 0;
}

int main(void) {
    install_fatal_handlers();
    install_term_handlers();
    maybe_allow_ptrace();
    linfo("main2: init begin tid=%d\n", gettid());
    // register the per-thread handler hook BEFORE crn_init: machine threads are
    // created during init, and only threads created afterwards run the callback
    // (otherwise a fiber stack overflow on those threads dies without markers)
    extern void initHook();
    initHook();
    linfo("main2: initHook done tid=%d\n", gettid());
    corona* nr0 = crn_new();
    linfo("main2: crn_new ok nr=%p\n", (void*)nr0);
    crn_set_thread_createcb(thread_create_cb, 0);
    crn_init(nr0);
    linfo("main2: crn_init returned tid=%d\n", gettid());
    crn_wait_init_done(nr0);
    linfo("main2: crn_wait_init_done ok tid=%d\n", gettid());

    g_seconds = env_i("CRN2_SECONDS", 3600);
    g_seed = env_u("CRN2_SEED", (unsigned)time(0));
    g_stats_sec = env_i("CRN2_STATS_SEC", 30);
    g_dns = env_i("CRN2_DNS", 1);
    g_deep = env_i("CRN2_DEEP", 0);
    g_overflow = env_i("CRN2_OVERFLOW", 0);
    g_abort_on_stall = env_i("CRN2_ABORT_ON_STALL", 0);
    g_destroy_on_exit = env_i("CRN2_DESTROY_ON_EXIT", 0);
    g_deadline_ms = now_ms() + (long)g_seconds * 1000;
    linfo("main2: deadline_at=%ld now=%ld\n", g_deadline_ms, now_ms());

    for (int i = 0; i < S_MAX; i++) {
        int on = scenario_selected(g_snames[i]);
        if (i == S10 && !g_dns) on = 0;
        if (i == S18 && !g_overflow) on = 0;
        if (i == S23 && count_fds() < 0) on = 0;
#ifndef __linux__
        if (i == S24) on = 0; // epoll scenario is Linux-only
#endif
        atomic_store(&g_enabled[i], on);
        atomic_store(&g_hb[i], now_ms());
    }
    linfo("main2 start sec=%d seed=%u stats=%d dns=%d deep=%d overflow=%d\n",
          g_seconds, g_seed, g_stats_sec, g_dns, g_deep, g_overflow);

    crn_post(controller, 0);
    linfo("main2: controller posted, waiting tid=%d\n", gettid());
    for (;;) {
        if (g_controller_done) break;
        if (now_ms() > g_deadline_ms + 10000) {
            linfo("controller stalled; hard exit %d\n", 0);
            _exit(1);
        }
        sleep(1);
    }
    linfo("main2: controller done rc=%d\n", g_controller_rc);
    return g_controller_rc;
}
