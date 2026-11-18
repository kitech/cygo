// t04: chan close semantics — parked fibers wake with failure, post-close ops
// fail fast (batch1 close lifecycle + batch3 closed-entry checks)
#include "t00_common.h"

static crn_hchan* done;

static void parked_recver(void* arg) {
    crn_hchan* ch = arg;
    void* v = (void*)0xdeadbeef;
    int rv = crn_hchan_recv(ch, &v);
    CHECK_EQ(rv, 0, "parked recver rv=%d", rv);
    CHECK(v == nilptr, "parked recver pdata=%p", v);
    t_join_send(done);
}

static void parked_sender(void* arg) {
    crn_hchan* ch = arg;
    int rv = crn_hchan_send(ch, (void*)42);
    CHECK_EQ(rv, 0, "parked sender rv=%d", rv);
    t_join_send(done);
}

static void controller(void* arg) {
    done = crn_hchan_new(0);

    // 1) three recv fibers park, then close
    crn_hchan* ch1 = crn_hchan_new(0);
    for (int i = 0; i < 3; i++) { crn_post(parked_recver, ch1); }
    t_usleep_ms(100); // let them park
    int rv = crn_hchan_close(ch1);
    CHECK_EQ(rv, 1, "close1 rv=%d", rv);
    CHECK(crn_hchan_is_closed(ch1), "closed flag");

    // 2) two send fibers park, then close
    crn_hchan* ch2 = crn_hchan_new(0);
    for (int i = 0; i < 2; i++) { crn_post(parked_sender, ch2); }
    t_usleep_ms(100);
    rv = crn_hchan_close(ch2);
    CHECK_EQ(rv, 1, "close2 rv=%d", rv);

    // 3) ops on already-closed chans return 0 immediately (no park)
    rv = crn_hchan_send(ch1, (void*)1);
    CHECK_EQ(rv, 0, "send after close rv=%d", rv);
    void* v = (void*)0xdeadbeef;
    rv = crn_hchan_recv(ch1, &v);
    CHECK_EQ(rv, 0, "recv after close rv=%d", rv);
    CHECK(v == nilptr, "recv after close pdata=%p", v);

    // 4) double close is a no-op returning true
    rv = crn_hchan_close(ch1);
    CHECK_EQ(rv, 1, "double close rv=%d", rv);

    // join: 3 recvers + 2 senders + 2 post-close ops happen inline above
    t_join_wait(done, 5);
    t_report_and_exit();
}

int main() {
    t_init();
    crn_post(controller, 0);
    for (;;) { sleep(1); }
}
