package builtin

/*
// #cgo LDFLAGS: -lcrn

#include <stdio.h>

extern int crn_post(voidptr, voidptr);
extern int crn_goid();

extern void* hchan_new(int);
extern void hchan_send(voidptr, voidptr);
extern void* hchan_recv(voidptr, voidptr);
*/
import "C"

///////////////////

//export crn_post
func crn_post(fn voidptr, arg voidptr) int {
	return 0
}

//export crn_goid
func crn_goid() int {
	return 0
}

//export hchan_new
func hchan_new(n int) voidptr {
	return 0
}

//export hchan_send
func hchan_send(ch voidptr, val voidptr) {
	return
}

//export hchan_recv
func hchan_recv(ch voidptr, val voidptr) voidptr {
	return 0
}

///////////////////

//export cxrt_fiber_post
func fiber_post(fnptr voidptr, arg voidptr) {
	id := C.crn_post(fnptr, arg)
	// return id
}

func fiberid() int {
	id := C.crn_goid()
	return id
}

//export cxrt_chan_new
func chan_new(sz int) voidptr {
	// return nil
	ch := C.hchan_new(sz)
	assert(ch != nil)
	C.printf("%s:%d cxrt_chan_new, %p %d\n", C.__FILE__, C.__LINE__, ch, sz)
	return ch
}

//export cxrt_chan_send
func chan_send(ch voidptr, arg voidptr) {
	assert(ch != nil)
	C.hchan_send(ch, arg)
}

//export cxrt_chan_recv
func chan_recv(ch voidptr) voidptr {
	assert(ch != nil)
	var data voidptr
	C.hchan_recv(ch, &data)
	return data
}
