// t03: buffered chan flood/drain with parking on full
#include "t00_common.h"

#define CAP 4
#define TOTAL 100
static crn_hchan* ch;
static crn_hchan* done;

static void producer(void* arg) {
    for (int i = 0; i < TOTAL; i++) {
        int rv = crn_hchan_send(ch, (void*)(uintptr_t)(i + 1));
        CHECK_EQ(rv, 1, "producer send rv=%d i=%d", rv, i);
        if ((i % 10) == 0) { t_usleep_ms(1); } // let consumer drain sometimes
    }
    t_join_send(done);
}

static void consumer(void* arg) {
    int last = 0;
    for (int i = 0; i < TOTAL; i++) {
        void* v = (void*)0xdeadbeef;
        int rv = crn_hchan_recv(ch, &v);
        CHECK_EQ(rv, 1, "consumer recv rv=%d i=%d", rv, i);
        int cur = (int)(uintptr_t)v;
        CHECK_EQ(cur, last + 1, "consumer order i=%d cur=%d", i, cur);
        last = cur;
        if ((i % 10) == 5) { t_usleep_ms(1); } // let producer fill up and park
    }
    CHECK_EQ(last, TOTAL, "consumer final last=%d", last);
    t_join_send(done);
}

static void controller(void* arg) {
    ch = crn_hchan_new(CAP);
    CHECK_EQ(crn_hchan_cap(ch), CAP, "cap");
    done = crn_hchan_new(0);
    crn_post(producer, 0);
    crn_post(consumer, 0);
    t_join_wait(done, 2);
    CHECK_EQ(crn_hchan_len(ch), 0, "chan drained len=%d", crn_hchan_len(ch));
    t_report_and_exit();
}

int main() {
    t_init();
    crn_post(controller, 0);
    for (;;) { sleep(1); }
}
