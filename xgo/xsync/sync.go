package xsync

/*
#include <pthread.h>
*/
import "C"

type Mutex struct {
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
