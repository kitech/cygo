package futex

/*
#include <pthread.h>

*/
import "C"
import "atomic"

// 空的 lock(), unlock
// with prof: mutex2 115 < mutex3 167ns/op  < mutex1 290ns/op
// no prof: mutex3 17ns/op < mutex2 27ns/op < mutex1 50ns/op


///

struct Mutex {
    impl1 *Mutex1// = 0
    impl2 *Mutex2// = 0
    impl3 *Mutex3// = 0
}

func newMutex() *Mutex {
    mu := &Mutex{}
    mu.impl1 = newMutex1()
    //mu.impl.impl2 = newMutex2()
    //mu.impl.impl3 = newMutex3()
    //mu.impl3 = newMutex3()
    return mu
}

func (mu *Mutex) mlock() {
    impl := mu.impl1
	impl.mlock()
    //mu.impl.impl2.mlock()
    //mu.impl.impl3.mlock()
    //mu.impl3.mlock()
}

//[inline]
func (mu *Mutex) munlock() {
    impl := mu.impl1
	impl.munlock()
    //mu.impl.impl2.munlock()
    //mu.impl.impl3.munlock()
    //mu.impl3.munlock()
}


///////////////////
// [ref_only]
struct Mutex1 {
    futval uint32
}

func newMutex1() *Mutex1 {
    mu := &Mutex1{}
    mu.futval = 0
    return mu
}

const (
    // unlocked = uint32(0) // TODO syntax error
	unlocked = 0
    locked = 1
    sleeping = 2
)

// oldval maybe modified
// [inline]
func cmpxchgu32(addr *uint32, oldval uint32, newval uint32) uint32 {
    ep := &oldval
    //C.atomic_compare_exchange_strong_u32(addr, ep, newval)
	atomic.CmpXchg32(addr, ep, newval)
    return *ep
}

func (this *Mutex1) mlock() {
    ov := uint32(0)
    {
        ov = cmpxchgu32(&this.futval, 0, 1)
        // println("$ov, ${this.futval}")
        if ov == 0 {
            // println("rettt")
            return
        }
    }
    for {
        if ov == 2 || cmpxchgu32(&this.futval, 1, 2) != 0 {
            futeximpl2(&this.futval, C.FUTEX_WAIT | C.FUTEX_PRIVATE_FLAG, 2)
        }
        ov = cmpxchgu32(&this.futval, 0, 2)
        if ov != 0 {
            C.syscall(C.__NR_sched_yield)
            continue
        }else{
            break
        }
    }
}

func (this *Mutex1) munlock() {
	if atomic.FetchSub32(&this.futval, 1) != 1{
		atomic.Store32(&this.futval, 0)
		futeximpl2(&this.futval, C.FUTEX_WAKE | C.FUTEX_PRIVATE_FLAG, 1)
	}
	/*
    if C.atomic_fetch_sub_u32(&this.futval, 1) != 1{
        C.atomic_store_u32(&this.futval, 0)
        futeximpl2(&this.futval, C.FUTEX_WAKE | C.FUTEX_PRIVATE_FLAG, 1)
    }
   */
}

// https://github.com/eliben/code-for-blog/blob/master/2018/futex-basics/mutex-using-futex.cpp

// 115ns/op
struct Mutex2 {
     val C.pthread_mutex_t
}

func newMutex2() *Mutex2 {
    mu := &Mutex2{}
    return mu
}
func (mu *Mutex2) mlock() {
    C.pthread_mutex_lock(&mu.val)
}
func (mu *Mutex2) munlock() {
    C.pthread_mutex_unlock(&mu.val)
}


struct Mutex3 {
    val C.pthread_spinlock_t
}

func newMutex3() *Mutex3 {
    mu := &Mutex3{}
    C.pthread_spin_init(&mu.val, 0)
    return mu
}

// //fn C.pthread_spin_init() int
// //fn C.pthread_spin_lock() int
// //fn C.pthread_spin_unlock() int

func (mu *Mutex3) mlock() {
    C.pthread_spin_lock(&mu.val)
}
func (mu *Mutex3) munlock() {
    C.pthread_spin_unlock(&mu.val)
}


