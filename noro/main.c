#include <unistd.h>
#include <stdio.h>
#include <assert.h>
#include <time.h>

#include <gc/gc.h>
#include <coro.h>
#include <collectc/hashtable.h>
#include <collectc/array.h>
#include "noro.h"
#include "norogc.h"

#define HKDEBUG 1
#define linfo(fmt, ...)                                                 \
    do { if (HKDEBUG) fprintf(stderr, "%s:%d:%s ", __FILE__, __LINE__, __FUNCTION__); } while (0); \
    do { if (HKDEBUG) fprintf(stderr, fmt, __VA_ARGS__); } while (0) ;


#include <sys/epoll.h>
#include "hook.h"
extern fcntl_t fcntl_f;
extern getsockopt_t getsockopt_f;
extern setsockopt_t setsockopt_f;
extern epoll_wait_t epoll_wait_f;

int noro_epoll_create() {
    int fd = epoll_create1(EPOLL_CLOEXEC);
    return fd;
}
int noro_epoll_wait(int epfd, struct epoll_event *events,
                     int maxevents, int timeout) {
    return epoll_wait_f(epfd, events, maxevents, timeout);
}

#include <unistd.h>
#include <sys/syscall.h>

pid_t gettid() {
#ifdef SYS_gettid
    pid_t tid = syscall(SYS_gettid);
    return tid;
#else
#error "SYS_gettid unavailable on this system"
    return 0;
#endif
}

void hello(void*arg) {
    int tid = gettid();
    linfo("called %p %d, %ld\n", arg, tid, time(0));
    // assert(1==2);
    for (int i = 0; i < 9; i++) {
        NORO_MALLOC(15550);
    }
    for (int i = 0; i < 1; i ++) {
        linfo("hello step. %d %d\n", i, tid);
        sleep(1);
        NORO_MALLOC(25550);
    }
    sleep(2);
    linfo("hello end %d %ld\n", tid, time(0)); // this tid not begin tid???
    assert(gettid() == tid);
}

static noro* nr;
int main() {
    nr = noro_new();
    noro_init(nr);
    noro_wait_init_done(nr);
    linfo("noro init done %d, %d\n", 12345, gettid());
    sleep(1);
    for (int i = 0; i < 3; i ++) {
        noro_post(hello, (void*)(uintptr_t)i);
    }
    for (;;) {
        for (int i = 0; i < 9; i ++) {
            NORO_MALLOC(35679);
        }
        noro_post(hello, (void*)(uintptr_t)5);
        socket(PF_INET, SOCK_STREAM, 0);
        linfo("before main sleep %d\n", time(0));
        sleep(1);
        linfo("after main sleep %d\n", time(0));
    }
    sleep(5);
    return 0;
}
