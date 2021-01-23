package epoll

/*
#include <fcntl.h>
#include <unistd.h>
#include <sys/types.h>
#include <sys/socket.h>
#include <netdb.h>
#include <sys/epoll.h>
#include <sys/timerfd.h>
#include <sys/eventfd.h>

#define ATEST_FLTVAL 1.23
*/
import "C"

/*
union Data {
    pub mut:
    ptr voidptr
    fd int
    u32val u32
    u64val u64
}
*/

struct Event {
    events uint32
    // data Data
	data uint64
}

/* TODO syntax */
/*
const (
    IN  = uint32(C.EPOLLIN )
    PRI = uint32(C.EPOLLPRI)
    OUT = uint32(C.EPOLLOUT)
    MSG = uint32(C.EPOLLMSG)
    ERR = uint32(C.EPOLLERR)
    HUP = uint32(C.EPOLLHUP)
    WAKEUP = uint32(C.EPOLLWAKEUP)
    ONESHOT = uint32(C.EPOLLONESHOT)
    ET = uint32(C.EPOLLET)
    EXCLUSIVE = uint32(C.EPOLLEXCLUSIVE)
)
*/

const (
	//_begin int = -1 // nolucky
    IN  = C.EPOLLIN
    PRI = C.EPOLLPRI
    OUT = C.EPOLLOUT
    MSG = C.EPOLLMSG
    ERR = C.EPOLLERR
    HUP = C.EPOLLHUP
    WAKEUP = C.EPOLLWAKEUP
    ONESHOT = C.EPOLLONESHOT
    ET = C.EPOLLET
    EXCLUSIVE = C.EPOLLEXCLUSIVE

	ONESHOT2 = 222
	//ONESHOT3 = int(C.EPOLLONESHOT) // not work
	//ONESHOT4 = (int)(C.EPOLLONESHOT) // not work

	AFLTVAL = 1.23// works
	AFLTVAL2 = C.ATEST_FLTVAL
)

// not work too
/*
var (
	WAKEUP = C.EPOLLWAKEUP
	ONESHOT = C.EPOLLONESHOT
)
*/

func tryprt_const() {
	foo := ONESHOT
	println(foo)
}

const (
    CTL_ADD = C.EPOLL_CTL_ADD
    CTL_DEL = C.EPOLL_CTL_DEL
    CTL_MOD = C.EPOLL_CTL_MOD
)

// fn C.epoll_create1(int) int
// fn C.epoll_ctl(epfd int, op int, fd int, event &Event) int
// fn C.epoll_wait(epfd int, events &Event, maxevents int, timeout int) int
// fn C.epoll_pwait(epfd int, events &Event, maxevents int, timeout int,
//              sigmask viodptr/*sigset_t*/) int


// int pipe(int pipefd[2]);
// int pipe2(int pipefd[2], int flags);
// fn C.pipe(pipefd &int) int
// fn C.pipe2(pipefd &int, flags int) int

// int eventfd(unsigned int initval, int flags);
// fn C.eventfd(initval u32, flags int) int


func wait(epfd int, events *Event, maxevents int, timeout int) int {
    return C.epoll_wait(epfd, events, maxevents, timeout)
}

