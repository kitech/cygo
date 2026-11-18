// t15: C -> Zig C-ABI demo functions (zigsrc/corona_zig.zig, linked into libcrn)
#include "t00_common.h"
#include "corona_zig.h"

static crn_hchan* done;

static void zig_user(void* arg) {
    CHECK_EQ(crn_zig_add(2, 3), 5, "crn_zig_add");
    CHECK_EQ(crn_zig_add(-7, 7), 0, "crn_zig_add zero");
    CHECK_EQ(crn_zig_fib(0), 0, "fib0");
    CHECK_EQ(crn_zig_fib(1), 1, "fib1");
    CHECK_EQ(crn_zig_fib(10), 55, "fib10");

    int32_t v[] = {1, 2, 3, 5};
    CHECK_EQ(crn_zig_sum(v, 4), 11, "sum");
    CHECK(crn_zig_is_sorted(v, 4), "sorted");
    int32_t w[] = {3, 1};
    CHECK(!crn_zig_is_sorted(w, 2), "unsorted");

    char buf[64] = {0};
    size_t n = crn_zig_banner(buf, sizeof(buf));
    CHECK(n > 0, "banner len=%zu", n);
    CHECK(strstr(buf, "corona-zig/") != NULL, "banner=%s", buf);

    const char* hello = crn_zig_hello();
    CHECK(hello != NULL && strcmp(hello, "hello from zig") == 0, "hello=%s", hello ? hello : "(null)");

    t_join_send(done);
}

static void controller(void* arg) {
    done = crn_hchan_new(0);
    crn_post(zig_user, 0);
    t_join_wait(done, 1);
    t_report_and_exit();
}

int main() {
    t_init();
    crn_post(controller, 0);
    for (;;) { sleep(1); }
}
