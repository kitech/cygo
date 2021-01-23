package iohook

/*
#cgo CFLAGS: -D_GNU_SOURCE

extern void iohook_initHook();
extern void iohook_initHook2();
*/
import "C"

func Keepme() {}

struct Yielder {
    // int (*incoro)();
    // void* (*getcoro)();
    // int (*yield)(long, int);
    // int (*yield_multi)(int, int, long*, int*);

	incoro voidptr
	getcoro voidptr
	yield voidptr
	yield_multi voidptr
}

func pre_main_init000(y voidptr, allocer voidptr) {
    // C.iohook_initHook(y, sizeof(rtcom.Yielder), allocer)
	C.iohook_initHook(y, 8*4, allocer)
}

func pre_main_init(incoro voidptr, getcoro voidptr, onyield voidptr, onyield_multi voidptr,
	mallocfn voidptr, callocfn voidptr, reallocfn voidptr, freefn voidptr) {
	C.iohook_initHook2(incoro , getcoro , onyield , onyield_multi ,
		mallocfn , callocfn , reallocfn , freefn )
}

func post_main_deinit() {

}



///////////////
// TODO enum???
const (
    YIELD_TYPE_NONE = 0
    YIELD_TYPE_CHAN_SEND =1
    YIELD_TYPE_CHAN_RECV =2
    YIELD_TYPE_CHAN_RECV_CLOSED =3
    YIELD_TYPE_CHAN_SELECT =4
    YIELD_TYPE_CHAN_SELECT_NOCASE =5
    YIELD_TYPE_CONNECT =6
    YIELD_TYPE_READ =7
    YIELD_TYPE_READV =8
    YIELD_TYPE_RECV =9
    YIELD_TYPE_RECVFROM = 10
    YIELD_TYPE_RECVMSG = 11
    YIELD_TYPE_RECVMSG_TIMEOUT = 12
    YIELD_TYPE_WRITE = 13
    YIELD_TYPE_WRITEV = 14
    YIELD_TYPE_SEND = 15
    YIELD_TYPE_SENDTO = 16
    YIELD_TYPE_SENDMSG = 17

    YIELD_TYPE_POLL = 18
    YIELD_TYPE_UUPOLL = 19 // __poll
    YIELD_TYPE_SELECT = 20
    YIELD_TYPE_ACCEPT = 21

    YIELD_TYPE_LOCK = 22
    YIELD_TYPE_TRYLOCK = 23
    YIELD_TYPE_UNLOCK  = 24
    YIELD_TYPE_COND_WAIT = 25
    YIELD_TYPE_COND_TIMEDWAIT = 25

    YIELD_TYPE_SLEEP = 27
    YIELD_TYPE_MSLEEP = 28
    YIELD_TYPE_USLEEP = 29
    YIELD_TYPE_NANOSLEEP = 30

    YIELD_TYPE_GETHOSTBYNAMER = 31
    YIELD_TYPE_GETHOSTBYNAME2R = 32
    YIELD_TYPE_GETHOSTBYADDR = 33
    YIELD_TYPE_GETADDRINFO = 34

    YIELD_TYPE_MAX = 35
)

