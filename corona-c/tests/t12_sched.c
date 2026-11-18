// t12: scheduler yield churn — crn_sched() storms and cross-fiber handoff
#include "t00_common.h"

#define NCHURN 2000
#define ROUNDS 300
static crn_hchan* done;
static crn_hchan* c1;
static crn_hchan* c2;

static void churner(void* arg) {
    for (int i = 0; i < NCHURN; i++) {
        crn_sched();
    }
    t_join_send(done);
}

static void handoff_pair(void* arg) {
    void** args = arg;
    crn_hchan* l = args[0];
    crn_hchan* r = args[1];
    int isb = (r == c1); // second fiber plays B: must not send on last round
    for (int i = 0; i < ROUNDS; i++) {
        void* v = (void*)0xdeadbeef;
        int rv = crn_hchan_recv(l, &v);
        CHECK_EQ(rv, 1, "handoff recv rv=%d i=%d", rv, i);
        if (isb && i == ROUNDS - 1) { break; }
        rv = crn_hchan_send(r, (void*)(uintptr_t)((uintptr_t)v + 1));
        CHECK_EQ(rv, 1, "handoff send rv=%d i=%d", rv, i);
    }
    t_join_send(done);
}

static void controller(void* arg) {
    done = crn_hchan_new(0);
    for (int i = 0; i < 4; i++) { crn_post(churner, 0); }

    // two fibers bouncing a counter over unbuffered chans
    static void* args1[2];
    static void* args2[2];
    c1 = crn_hchan_new(0);
    c2 = crn_hchan_new(0);
    args1[0] = c1; args1[1] = c2;
    args2[0] = c2; args2[1] = c1;
    crn_post(handoff_pair, args1);
    crn_post(handoff_pair, args2);
    int rv = crn_hchan_send(c1, (void*)0); // kick
    CHECK_EQ(rv, 1, "handoff kick rv=%d", rv);

    t_join_wait(done, 4 + 2);
    t_report_and_exit();
}

int main() {
    t_init();
    crn_post(controller, 0);
    for (;;) { sleep(1); }
}
