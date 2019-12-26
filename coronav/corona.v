module coronav

#flag -I@VMOD/cxrt/corona-c -I@VMOD/cxrt/src -I@VMOD/cxrt/cltc/include
#flag -L@VMOD/cxrt/bysrc
#flag -L@VMOD/cxrt/cltc/lib
#flag -lcxrt -lcrn -l:libcollectc.a -levent -levent_pthreads
#flag -L@VMOD/cxrt/bdwgc/.libs -lgc -lpthread

#include "corona_util.h"
#include "crnpub.h"
#include "cxrtbase.h"

// struct C.corona{}

pub struct Corona {
mut:
    h *C.corona
	useit int
}

fn C.crn_init_and_wait_done() *C.corona
fn C.crn_goid() int
fn C.crn_post(f voidptr, arg voidptr)

fn C.gettid() int
// fn C.sleep(s int) int

pub fn new() &Corona{
	crn := &Corona{0,0}
	h := crn_init_and_wait_done()
	println('h=$h')
	return crn
}

pub fn (crn mut Corona) post(f fn (voidptr), arg voidptr) {
	crn.useit++
	C.crn_post(f, arg)
}

pub fn (crn mut Corona) goid() int {
	crn.useit++
	return C.crn_goid()
}

pub fn goid() int { return C.crn_goid() }

pub fn gettid() int { return C.gettid() }
pub fn sleep(s int) { C.sleep(s) }

pub fn post(f fn(voidptr), arg voidptr) {
	C.crn_post(f, arg)
}

fn C.cxrt_init_env()

pub fn init_env() {
	//
	C.cxrt_init_env()
}
