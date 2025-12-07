#include <unistd.h>
#include <stdio.h>
#include <assert.h>
#include <time.h>

#include <gc.h>
#include <coro.h>
#include <collectc/hashtable.h>
#include <collectc/array.h>
#include "corona.h"
#include "coronagc.h"
#include "coronapriv.h"
#include "futex.h"
#include "rxilog.h"
#include "corona_util.h"

#include "hook.h"
extern fcntl_t fcntl_f;
extern getsockopt_t getsockopt_f;
extern setsockopt_t setsockopt_f;

#ifdef __APPLE__
#else
#include <sys/epoll.h>
extern epoll_wait_t epoll_wait_f;

int crn_epoll_create() {
    int fd = epoll_create1(EPOLL_CLOEXEC);
    return fd;
}
int crn_epoll_wait(int epfd, struct epoll_event *events,
                     int maxevents, int timeout) {
    return epoll_wait_f(epfd, events, maxevents, timeout);
}
#endif

void hello(void*arg) {
    int tid = gettid();
    linfo("called %d %d, %ld\n", arg, tid, time(0));
    // assert(1==2);
    for (int i = 0; i < 9; i++) {
        crn_gc_malloc(15550);
    }
    for (int i = 0; i < 1; i ++) {
        linfo("hello(%d) step. %d %d\n", arg, i, tid);
        sleep(1);
        crn_gc_malloc(25550);
    }
    sleep(2);
    linfo("hello(%d) end %d %ld\n", arg, tid, time(0)); // this tid not begin tid???
    assert(gettid() == tid);
}

static corona* nr = nilptr;
int main() {
    extern void initHook(); initHook();
    nr = crn_new();
    crn_init(nr);
    crn_wait_init_done(nr);
    linfo("corona init done %d, %d\n", 12345, gettid());
    sleep(1);
    for (int i = 0; i < 3; i ++) {
        crn_post(hello, (void*)(uintptr_t)(i+1));
    }
    // seems there is race condition when strart up, and malloc big size object below.
    // so collectc once with lucky
    GC_gcollect();
    for (int j=100;;j++) {
        for (int i = 0; i < 9; i ++) {
            crn_gc_malloc(35679);
        }
        crn_post(hello, (void*)(uintptr_t)j);
        int fd = socket(PF_INET, SOCK_STREAM, 0);
        assert(fd>0);
        sleep(1);
        close(fd);
    }
    sleep(5);
    return 0;
}
