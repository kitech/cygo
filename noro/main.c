#include <unistd.h>
#include <stdio.h>

#include <gc/gc.h>
#include <coro.h>
#include <collectc/hashtable.h>
#include <collectc/array.h>
#include "noro.h"

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

void hello(void*arg) {
    linfo("called %p\n", arg);
}

static noro* nr;
int main() {
    nr = noro_new();
    noro_init(nr);
    noro_wait_init_done(nr);
    linfo("noro init done %d\n", 1);
    sleep(1);
    noro_post(&hello, 0);
    sleep(5);
    return 0;
}
