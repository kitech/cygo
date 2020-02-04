package xsync

/*
#include <pthread.h>
*/
import "C"

func init() {
	if true {
		assert(sizeof(Mutex) == sizeof(C.pthread_mutex_t))
	}
}

type Mutex struct {
	// TODO compilerd to voidptr lock, and failed then
	// if in somewhere have use of C.pthread_mutex_t, then it's works again
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
