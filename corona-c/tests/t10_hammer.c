// t10: many-to-one chan hammer — 10 senders, 1 consumer, exact totals
#include "t00_common.h"

#define NSND 10
#define EACH 100
static crn_hchan* ch;
static crn_hchan* done;

static void sender(void* arg) {
    int base = (int)(uintptr_t)arg;
    for (int i = 0; i < EACH; i++) {
        int rv = crn_hchan_send(ch, (void*)(uintptr_t)(base * 1000 + i));
        CHECK_EQ(rv, 1, "sender send rv=%d base=%d i=%d", rv, base, i);
    }
    t_join_send(done);
}

static void consumer(void* arg) {
    int got[NSND] = {0};
    for (int i = 0; i < NSND * EACH; i++) {
        void* v = (void*)0xdeadbeef;
        int rv = crn_hchan_recv(ch, &v);
        CHECK_EQ(rv, 1, "consumer recv rv=%d i=%d", rv, i);
        int val = (int)(uintptr_t)v;
        int base = val / 1000;
        int idx = val % 1000;
        CHECK(base >= 0 && base < NSND, "consumer val=%d", val);
        CHECK_EQ(idx, got[base], "consumer order val=%d i=%d", val, i);
        got[base]++;
    }
    for (int b = 0; b < NSND; b++) {
        CHECK_EQ(got[b], EACH, "sender %d got=%d", b, got[b]);
    }
    t_join_send(done);
}

static void controller(void* arg) {
    ch = crn_hchan_new(8);
    done = crn_hchan_new(0);
    crn_post(consumer, 0);
    for (int i = 0; i < NSND; i++) { crn_post(sender, (void*)(uintptr_t)i); }
    t_join_wait(done, NSND + 1);
    CHECK_EQ(crn_hchan_len(ch), 0, "ring drained len=%d", crn_hchan_len(ch));
    t_report_and_exit();
}

int main() {
    t_init();
    crn_post(controller, 0);
    for (;;) { sleep(1); }
}
