// t08: fd/EAGAIN yield loop + optional DNS (CRN_TEST_DNS=1)
// NOTE: iohook's fd tracking is compiled out (iohookcb registers nothing), so a
// plain socketpair recv returns EAGAIN instead of parking in the netpoller.
// This test verifies the (real) EAGAIN+yield loop behaviour instead.
#include "t00_common.h"
#include <sys/socket.h>
#include <fcntl.h>
#include <errno.h>

static crn_hchan* done;

static void set_nb(int fd) {
    int fl = fcntl(fd, F_GETFL, 0);
    fcntl(fd, F_SETFL, fl | O_NONBLOCK);
}

static void sock_reader(void* arg) {
    void** args = arg;
    int fd = (int)(uintptr_t)args[0];
    char buf[64] = {0};
    int tries = 0;
    for (;;) {
        int rv = recv(fd, buf, sizeof(buf) - 1, 0);
        if (rv == 5) { break; }
        if (rv < 0 && (errno == EAGAIN || errno == EWOULDBLOCK || errno == EINTR)) {
            tries++;
            CHECK(tries < 20000, "reader never got data, tries=%d", tries);
            usleep(1000); // yield 1ms
            continue;
        }
        CHECK(0, "recv unexpected rv=%d errno=%d", rv, errno);
        break;
    }
    CHECK(memcmp(buf, "hello", 5) == 0, "recv buf=%.5s", buf);
    t_join_send(done);
}

static void dns_user(void* arg) {
    extern char* crn_getaddrinfo(const char* hostname);
    for (int i = 0; i < 10; i++) {
        char* ip = crn_getaddrinfo("localhost");
        CHECK(ip != nilptr, "dns localhost iter=%d", i);
    }
    t_join_send(done);
}

static void controller(void* arg) {
    done = crn_hchan_new(0);
    int njoin = 0;

    int sv[2];
    int rv = socketpair(AF_UNIX, SOCK_STREAM, 0, sv);
    CHECK_EQ(rv, 0, "socketpair");
    set_nb(sv[0]);
    static void* args[1];
    args[0] = (void*)(uintptr_t)sv[0];
    crn_post(sock_reader, args);
    njoin++;
    t_usleep_ms(50); // let the reader start spinning on EAGAIN
    rv = send(sv[1], "hello", 5, 0);
    CHECK_EQ(rv, 5, "kick send rv=%d", rv);

    if (getenv("CRN_TEST_DNS") != nilptr) {
        crn_post(dns_user, 0);
        njoin++;
    }

    t_join_wait(done, njoin);
    t_report_and_exit();
}

int main() {
    t_init();
    crn_post(controller, 0);
    for (;;) { sleep(1); }
}
