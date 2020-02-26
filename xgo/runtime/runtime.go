package runtime

// memory, thread, libc

/*
#include <stdlib.h>
#include <stdio.h>
#include <pthread.h>
#include <gc.h>

static void cygo_pmutex_foo() {}

typedef int (*pmutex_lock_t)(pthread_mutex_t *mutex);
extern pmutex_lock_t pmutex_lock_f;
typedef int (*pmutex_unlock_t)(pthread_mutex_t *mutex);
extern pmutex_unlock_t pmutex_unlock_f;

void cygo_pmutex_lock(voidptr mu) { pmutex_lock_f(mu); }
void cygo_pmutex_unlock(voidptr mu) { pmutex_unlock_f(mu); }

typedef int (*pcond_wait_t)(pthread_cond_t *restrict cond,
                      pthread_mutex_t *restrict mutex);
extern pcond_wait_t pcond_wait_f;
typedef int (*pcond_signal_t)(pthread_cond_t *cond);
extern pcond_signal_t pcond_signal_f;
typedef int (*pcond_broadcast_t)(pthread_cond_t *cond);
extern pcond_broadcast_t pcond_broadcast_f;

void cygo_pcond_wait(voidptr cd, voidptr mu) { pcond_wait_f(cd, mu); }
void cygo_pcond_signal(voidptr cd) { pcond_signal_f(cd); }
void cygo_pcond_broadcast(voidptr cd) { pcond_broadcast_f(cd); }

int cygo_pthread_create(voidptr h, voidptr fn, voidptr arg) {
   return GC_pthread_create(h, 0, fn, arg);
}
int cygo_pthread_join(pthread_t h, voidptr* retval) {
   return GC_pthread_join(h, retval);
}
int cygo_pthread_detach(pthread_t h) { return GC_pthread_detach(h); }
*/
import "C"

// it's realy raw mutex, without hook, internal use
type pmutex struct {
	obj C.pthread_mutex_t
}

func newpmutex() *pmutex {
	mu := &pmutex{}
	return mu
}
func (mu *pmutex) foo() {
	// C.cygo_pmutex_foo() // TODO compiler cparser
}
func (mu *pmutex) lock()   { C.cygo_pmutex_lock(&mu.obj) }
func (mu *pmutex) unlock() { C.cygo_pmutex_unlock(&mu.obj) }

///
type pcond struct {
	obj C.pthread_cond_t
}

func newpcond() *pcond {
	cd := &pcond{}
	return cd
}

func (cd *pcond) wait(mu *pmutex) {
	C.cygo_pcond_wait(&cd.obj, &mu.obj)
}
func (cd *pcond) signal() {
	C.cygo_pcond_signal(&cd.obj)
}
func (cd *pcond) broadcast(mu *pmutex) {
	C.cygo_pcond_broadcast(&cd.obj)
}

///
type pthread struct {
	handle C.pthread_t
	state  int
	mu     *pmutex
	cd     *pcond

	fnptr func(arg *pthread) voidptr
	fnarg voidptr
}

func newpthread(fnptr func(arg *pthread) voidptr, fnarg voidptr) *pthread {
	thr := &pthread{}
	thr.fnptr = fnptr
	thr.fnarg = fnarg

	thr.mu = newpmutex()
	thr.cd = newpcond()
	return thr
}

// not used, just demo
func pthread_proc_demo(thr *pthread) voidptr {
	// rv := thr.fnptr(thr.fnarg) // TODO compiler
	fnptr := thr.fnptr
	rv := fnptr(thr.fnarg)
	return rv
}

func (thr *pthread) start() {
	thr.mu.lock()
	state := thr.state
	if state == 0 {
		thr.state = 1
	}
	thr.mu.unlock()
	if state > 0 {
		println("pthread already started")
		return
	}
	rv := C.cygo_pthread_create(&thr.handle, thr.fnptr, thr)
}
func (thr *pthread) getarg() voidptr { return thr.fnarg }

func (thr *pthread) join() int {
	rv := C.cygo_pthread_join(thr.handle, nil)
	thr.state = 0
	return rv
}
func (thr *pthread) detach() int {
	rv := C.cygo_pthread_detach(thr.handle)
	return rv
}

// only callable in thr.fnptr
func (thr *pthread) suspend() {
	rv := C.pthread_self()
	if rv != thr.handle {
		println("cannot suspend thread in other thread")
		return
	}

	thr.mu.lock()
	thr.state++
	thr.cd.wait(thr.mu)
	thr.state--
	thr.mu.unlock()
}

// not callable in thr.fnptr
func (thr *pthread) resume() {
	rv := C.pthread_self()
	if rv == thr.handle {
		println("cannot resume thread in same thread")
		return
	}

	thr.mu.lock()
	state := thr.state
	if state == 2 {
		thr.cd.signal()
	} else if state == 1 {
		println("pthread not suspended")
		thr.cd.signal()
	} else {
		println("pthread not started")
	}
	thr.mu.unlock()
}

func Keep() {}
