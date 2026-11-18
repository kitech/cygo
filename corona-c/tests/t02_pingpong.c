// t02: ping-pong rendezvous over unbuffered chans (libgoc pattern)
#include "t00_common.h"

#define ROUNDS 1000
static crn_hchan* a2b;
static crn_hchan* b2a;
static crn_hchan* done;

static void player(void* arg) {
    int isb = (arg == (void*)2);
    crn_hchan* recv = isb ? b2a : a2b;
    crn_hchan* send = isb ? a2b : b2a;
    for (int i = 0; i < ROUNDS; i++) {
        void* v = (void*)0xdeadbeef;
        int rv = crn_hchan_recv(recv, &v);
        CHECK_EQ(rv, 1, "player recv rv=%d i=%d", rv, i);
        if ((i % 100) == 0) { linfo("player%c round %d\n", isb ? 'B' : 'A', i); }
        uintptr_t want = (uintptr_t)(2 * i + (isb ? 1 : 0));
        CHECK_EQ((uintptr_t)v, want, "player val i=%d v=%p want=%p", i, v, (void*)want);
        if (isb && i == ROUNDS - 1) { break; } // last B send would have no receiver
        rv = crn_hchan_send(send, (void*)(uintptr_t)((uintptr_t)v + 1));
        CHECK_EQ(rv, 1, "player send rv=%d i=%d", rv, i);
    }
    t_join_send(done);
}

static void controller(void* arg) {
    a2b = crn_hchan_new(0);
    b2a = crn_hchan_new(0);
    done = crn_hchan_new(0);
    int id = crn_post(player, (void*)1);
    CHECK(id > 0, "post player1");
    id = crn_post(player, (void*)2);
    CHECK(id > 0, "post player2");
    // kick off the exchange
    int rv = crn_hchan_send(a2b, (void*)(uintptr_t)0);
    CHECK_EQ(rv, 1, "kick send rv=%d", rv);
    t_join_wait(done, 2);
    t_report_and_exit();
}

int main() {
    t_init();
    crn_post(controller, 0);
    for (;;) { sleep(1); }
}
