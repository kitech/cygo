// t05: select semantics — default, two-chan race, close-wake, closed chans
#include "t00_common.h"

static crn_hchan* done;

// crn_scase is opaque in hchan.h; mirror the layout from hselect.c so the
// test can read the received element after select returns
typedef struct t_scase_mirror {
    crn_hchan* hc;
    uint16_t kind;
    void* hcelem;
    crn_hcdata* hcdt;
    uintptr_t pc;
    int64_t reltime;
} t_scase_mirror;

static void select_then_close_recv(void* arg) {
    void** args = arg;
    crn_hchan* ch = args[0];
    crn_hchan* donec = args[1];
    crn_scase* cas0[1];
    cas0[0] = crn_scase_new(ch, caseRecv, nilptr);
    int casi = -1;
    bool ok = crn_goselect(&casi, cas0, 1); // parks on recv
    // woken by close: recv reports EOF
    CHECK_EQ(ok, false, "select close-wake ok=%d", ok);
    CHECK_EQ(casi, 0, "select close-wake casi=%d", casi);
    crn_scase_free(cas0[0]);
    t_join_send(donec);
}

static void select_then_close_send(void* arg) {
    void** args = arg;
    crn_hchan* ch = args[0];
    crn_hchan* donec = args[1];
    crn_scase* cas0[1];
    cas0[0] = crn_scase_new(ch, caseSend, (void*)7);
    int casi = -1;
    bool ok = crn_goselect(&casi, cas0, 1); // parks on send
    // woken by close: send reports failure
    CHECK_EQ(ok, false, "select close-wake send ok=%d", ok);
    crn_scase_free(cas0[0]);
    t_join_send(donec);
}

static void two_recv_selector(void* arg) {
    void** args = arg;
    crn_hchan* c1 = args[0];
    crn_hchan* c2 = args[1];
    crn_hchan* donec = args[2];
    crn_scase* cas0[2];
    cas0[0] = crn_scase_new(c1, caseRecv, nilptr);
    cas0[1] = crn_scase_new(c2, caseRecv, nilptr);
    int casi = -1;
    bool ok = crn_goselect(&casi, cas0, 2); // parks on both
    CHECK_EQ(ok, true, "race select ok=%d", ok);
    CHECK(casi == 0 || casi == 1, "race select casi=%d", casi);
    t_scase_mirror* mk = (t_scase_mirror*)cas0[casi];
    CHECK(mk->hcelem != nilptr, "race select elem=%p", mk->hcelem);
    crn_scase_free(cas0[0]);
    crn_scase_free(cas0[1]);
    t_join_send(donec);
}

static void sender_to(void* arg) {
    crn_hchan* ch = arg;
    t_usleep_ms(50);
    int rv = crn_hchan_send(ch, (void*)42);
    CHECK_EQ(rv, 1, "sender rv=%d", rv);
    t_join_send(done);
}

static void closer(void* arg) {
    crn_hchan* ch = arg;
    t_usleep_ms(50);
    int rv = crn_hchan_close(ch);
    CHECK_EQ(rv, 1, "closer rv=%d", rv);
    t_join_send(done);
}

static void controller(void* arg) {
    done = crn_hchan_new(0);
    static void* argsA[3];
    static void* argsB[3];
    static void* argsC[3];

    // 1) select+default on empty chan: immediate, default picked
    {
        crn_hchan* ch = crn_hchan_new(4);
        crn_scase* cas0[2];
        cas0[0] = crn_scase_new(ch, caseRecv, nilptr);
        cas0[1] = crn_scase_new(ch, caseDefault, nilptr);
        int casi = -1;
        bool ok = crn_goselect(&casi, cas0, 2);
        CHECK_EQ(ok, false, "default ok=%d", ok);
        CHECK_EQ(casi, 1, "default casi=%d", casi);
        crn_scase_free(cas0[0]);
        crn_scase_free(cas0[1]);
    }

    // 2) blocked recv-select woken by close
    {
        crn_hchan* ch = crn_hchan_new(0);
        argsA[0] = ch; argsA[1] = done;
        crn_post(select_then_close_recv, argsA);
        crn_post(closer, ch);
    }

    // 3) blocked send-select woken by close
    {
        crn_hchan* ch = crn_hchan_new(0);
        argsB[0] = ch; argsB[1] = done;
        crn_post(select_then_close_send, argsB);
        crn_post(closer, ch);
    }

    // 4) two-chan race: exactly one wins, casi matches
    {
        crn_hchan* c1 = crn_hchan_new(0);
        crn_hchan* c2 = crn_hchan_new(0);
        argsC[0] = c1; argsC[1] = c2; argsC[2] = done;
        crn_post(two_recv_selector, argsC);
        crn_post(sender_to, c1);
    }

    // join: 1 closer/selector pair x3 + sender + selector = 5 fibers
    t_join_wait(done, 5);
    t_report_and_exit();
}

int main() {
    t_init();
    crn_post(controller, 0);
    for (;;) { sleep(1); }
}
