// t09: mixed stress — random fibers doing timer/mutex/GC/chan work for N sec
#include "t00_common.h"

#define NOPS 20
static crn_hchan* done;
static crn_mutex* mu;
static long counter;
static long deadline_ms;

static long now_ms() {
    struct timespec ts;
    clock_gettime(CLOCK_MONOTONIC, &ts);
    return ts.tv_sec * 1000 + ts.tv_nsec / 1000000;
}

static void stress_worker(void* arg) {
    unsigned seed = (unsigned)(uintptr_t)arg ^ (unsigned)time(0);
    for (int i = 0; i < NOPS; i++) {
        int op = rand_r(&seed) % 5;
        if (op == 0) {
            usleep(1 + (seed % 5) * 1000); // timer yield
        } else if (op == 1) {
            crn_mutex_lock(mu);
            counter++;
            crn_mutex_unlock(mu);
        } else if (op == 2) {
            crn_gc_malloc(2048);
        } else if (op == 3) {
            // self-contained buffered chan round-trip: never blocks
            crn_hchan* ch = crn_hchan_new(4);
            int rv = crn_hchan_send(ch, (void*)1);
            CHECK_EQ(rv, 1, "stress send rv=%d", rv);
            void* v = nilptr;
            rv = crn_hchan_recv(ch, &v);
            CHECK_EQ(rv, 1, "stress recv rv=%d", rv);
            CHECK_EQ((uintptr_t)v, 1, "stress recv val=%p", v);
            crn_hchan_close(ch);
        } else {
            crn_sched();
        }
    }
    t_join_send(done);
}

static void controller(void* arg) {
    int seconds = arg != nilptr ? (int)(uintptr_t)arg : 5;
    done = crn_hchan_new(0);
    mu = crn_mutex_new();
    deadline_ms = now_ms() + seconds * 1000;

    int posted = 0;
    while (now_ms() < deadline_ms - 500 && posted < 200) {
        crn_post(stress_worker, (void*)(uintptr_t)(posted + 1));
        posted++;
        usleep(2 * 1000);
    }
    linfo("stress posted=%d %d\n", posted, 0);
    t_join_wait(done, posted);
    linfo("stress counter=%ld %d\n", counter, 0);
    t_report_and_exit();
}

int main(int argc, char** argv) {
    t_init();
    crn_post(controller, (void*)(uintptr_t)(argc > 1 ? atoi(argv[1]) : 5));
    for (;;) { sleep(1); }
}
