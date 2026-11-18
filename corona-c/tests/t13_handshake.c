// t13: unbuffered direct-consume paths — parked sender met by fresh receiver
// and vice versa (exercises crn_hcdata_woke_set + resume ownership fix)
#include "t00_common.h"

static crn_hchan* done;

static void late_recver(void* arg) {
    void** args = arg;
    crn_hchan* ch = args[0];
    crn_hchan* donec = args[1];
    t_usleep_ms(50); // let the sender park first
    void* v = (void*)0xdeadbeef;
    int rv = crn_hchan_recv(ch, &v);
    CHECK_EQ(rv, 1, "late recver rv=%d", rv);
    CHECK_EQ((uintptr_t)v, 99, "late recver val=%p", v); // parked_sender sends 99
    t_join_send(donec);
}

static void late_sender(void* arg) {
    void** args = arg;
    crn_hchan* ch = args[0];
    crn_hchan* donec = args[1];
    t_usleep_ms(50); // let the recver park first
    int rv = crn_hchan_send(ch, (void*)42);
    CHECK_EQ(rv, 1, "late sender rv=%d", rv);
    t_join_send(donec);
}

static void parked_sender(void* arg) {
    void** args = arg;
    crn_hchan* ch = args[0];
    crn_hchan* donec = args[1];
    int rv = crn_hchan_send(ch, (void*)99); // parks immediately
    CHECK_EQ(rv, 1, "parked sender rv=%d", rv);
    t_join_send(donec);
}

static void parked_recver(void* arg) {
    void** args = arg;
    crn_hchan* ch = args[0];
    crn_hchan* donec = args[1];
    void* v = (void*)0xdeadbeef;
    int rv = crn_hchan_recv(ch, &v); // parks immediately
    CHECK_EQ(rv, 1, "parked recver rv=%d", rv);
    CHECK_EQ((uintptr_t)v, 42, "parked recver val=%p", v); // late_sender sends 42
    t_join_send(donec);
}

static void controller(void* arg) {
    done = crn_hchan_new(0);
    static void* args1[2];
    static void* args2[2];

    // 1) sender parks (no recver), recver arrives later
    {
        crn_hchan* ch = crn_hchan_new(0);
        args1[0] = ch; args1[1] = done;
        crn_post(parked_sender, args1);
        crn_post(late_recver, args1);
    }

    // 2) recver parks (no sender), sender arrives later
    {
        crn_hchan* ch = crn_hchan_new(0);
        args2[0] = ch; args2[1] = done;
        crn_post(parked_recver, args2);
        crn_post(late_sender, args2);
    }

    t_join_wait(done, 4);
    t_report_and_exit();
}

int main() {
    t_init();
    crn_post(controller, 0);
    for (;;) { sleep(1); }
}
