package xsync

/*
#include <pthread.h>
*/
import "C"

func Keep() {}

func init() {
	if true {
		// impl1 // TODO
		// sz1 := sizeof(Mutex{}) // TODO compiler
		sz1 := sizeof(Mutex) // TODO compiler
		pthmu := &C.pthread_mutex_t{}
		// assert(sz1 == sizeof(C.pthread_mutex_t)) // TODO compiler
		assert(sz1 == sizeof(*pthmu))

		// impl2
		// sz2 := unsafe.Sizeof(Mutex{}) // TODO compiler
		// assert(sz2 == sizeof(C.pthread_mutex_t))

		// impl3
		// sz1 := unsafe.Sizeof(Mutex{})  // TODO compiler
		// sz2 := unsafe.Sizeof(C.pthread_mutex_t{})  // TODO compiler
	}
}

type Mutex struct {
	// TODO compilerd to voidptr lock, and failed then
	// if in somewhere have use of C.pthread_mutex_t, then it's works again
	// oh, it is a union
	lock C.pthread_mutex_t
}

func (mu *Mutex) Lock() {
	C.pthread_mutex_lock(&mu.lock)
}
func (mu *Mutex) Unlock() {
	C.pthread_mutex_unlock(&mu.lock)
}

type Once struct {
	did int
}

func (once *Once) Do( /* todo f func()*/ ) {

}
