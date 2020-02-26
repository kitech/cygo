package runtime

// memory, thread, libc

/*
#include <stdlib.h>
#include <stdio.h>
#include <pthread.h>

*/
import "C"

type Thread struct {
	handle C.pthread_t
	state  int

	fnptr func(arg voidptr) voidptr
	fnarg voidptr
}

func NewThread(fnptr func(arg voidptr) voidptr, fnarg voidptr) *Thread {
	thr := &Thread{}
	thr.fnptr = fnptr
	thr.fnarg = fnarg

	return thr
}

func (thr *Thread) start() {

}

func (thr *Thread) suspend() {

}

func (thr *Thread) resume() {

}

func keep() {}
