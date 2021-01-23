package futex

/*
#cgo CFLAGS: -D_GNU_SOURCE

#include <stdio.h>
#include <errno.h>
#include <stdatomic.h>
#include <stdint.h>
#include <stdlib.h>
#include <unistd.h>
#include <sys/wait.h>
#include <sys/mman.h>
#include <sys/syscall.h>
#include <linux/futex.h>
#include <stdint.h>
#include <sys/time.h>
*/
import "C"
import "atomic"

func usemod() {
	// mu := sync.new_rwmutex()
}

func Keepme() {}

// [ref_only]
struct Futex {
    futval uint32
}

//long futex(uint32_t *uaddr, int futex_op, uint32_t val,
//           const struct timespec *timeout,   /* or: uint32_t val2 */
//         uint32_t *uaddr2, uint32_t val3);
//

// fn C.syscall() int

// const vnil = voidptr(0)

func newFutex() *Futex {
    this := &Futex{}
    this.futval = 0 // first wait
    return this
}

// [inline]
func futeximpl(uaddr *uint32, futex_op int, val uint32,
                 timeout voidptr, uaddr2 *uint32, val3 uint32) int {
    return C.syscall(C.SYS_futex, uaddr, futex_op, val,
		timeout, uaddr2, val3)
}

// [inline]
func futeximpl2(uaddr *uint32, futex_op int, val uint32) int {
    return C.syscall(C.SYS_futex, uaddr, futex_op, val, nil, nil, 0)
}

// vlib/sync/channels.v
// fn C.atomic_compare_exchange_strong_u32(&u32, &u32, u32) bool

func (this *Futex) wait() int {
    for {
        spinok := false
        for i in 0..30 {
            one := uint32(1)
            //ok := C.atomic_compare_exchange_strong_u32(&this.futval, &one, 0)
			ok := atomic.CmpXchg32(&this.futval, &one,0)
            if ok {
                spinok = true
                break
            }
            // yield
            C.syscall(C.__NR_sched_yield)//
            // https://chromium.googlesource.com/chromium/src/third_party/WebKit/Source/wtf/+/823d62cdecdbd5f161634177e130e5ac01eb7b48/SpinLock.cpp
            // asm { pause }
        }
        if spinok { break }

        // println("waiting...")
        rv := futeximpl(&this.futval, C.FUTEX_WAIT | C.FUTEX_PRIVATE_FLAG, 0, nil, nil, 0)
        eno := C.errno
        if rv == -1 && C.errno != C.EAGAIN {
            panic("futext-FUTEX_WAIT")
        }
        if rv == -1 {
            // println("wtt $eno")
        }
    }
    return 0
}
func (this *Futex) park() int {
	return this.wait()
}

func (this *Futex) wake() int {
    zero := (uint32)(0)

    //ok := C.atomic_compare_exchange_strong_u32(&this.futval, &zero, 1)
	ok := atomic.CmpXchg32(&this.futval, &zero, 1)
    if ok {
        rv := futeximpl(&this.futval, C.FUTEX_WAKE | C.FUTEX_PRIVATE_FLAG, 1, nil, nil, 0)
        if rv == -1 {
            panic("futex-FUTEX_WAKE")
        }
        return rv
    }
    return 0
}

