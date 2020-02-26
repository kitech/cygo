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

/// hook and yield public use
type Mutex struct {
	obj C.pthread_mutex_t
}

func NewMutex() *Mutex {
	mu := &Mutex{}
	return mu
}
func (mu *Mutex) lock()   { C.pthread_mutex_lock(&mu.obj) }
func (mu *Mutex) unlock() { C.pthread_mutex_unlock(&mu.obj) }

type Cond struct {
	obj C.pthread_cond_t
}

func NewCond() *Cond {
	cd := &Cond{}
	return cd
}
func (cd *Cond) wait(mu *Mutex) {
	C.pthread_cond_wait(&cd.obj, &mu.obj)
}
func (cd *Cond) signal() {
	C.pthread_cond_signal(&cd.obj)
}
func (cd *Cond) broadcast(mu *Mutex) {
	C.pthread_cond_broadcast(&cd.obj)
}

type Once struct {
	did int
}

func (once *Once) Do( /* todo f func()*/ ) {

}
