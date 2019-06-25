module corona

#flag -I $HOME/code/corona/corona-c
#flag -L $HOME/me/code/corona/corona-c  -L $HOME/code/corona/cltc/lib -lcorona -l:libcollectc.a -levent -levent_pthreads -L $HOME/code/corona/bdwgc/.libs -lgc -lpthread

#include <crnpub.h>

struct C.corona{}

struct Corona{
    h *C.corona
}

fn C.crn_init_and_wait_done() *C.corona
fn C.crn_get_goid() int
fn C.crn_post(f voidptr, arg voidptr)

fn C.sleep(s int) int

pub fn new() *Corona{
   crn := &Corona{}
   h := crn_init_and_wait_done()
   return crn
}

pub fn (crn mut Corona) post(f voidptr, arg voidptr) {
  C.crn_post(f, arg)
}

pub fn (crn mut Corona) goid() int {
   return C.crn_get_goid()
}

pub fn goid() int { return C.crn_get_goid() }

pub fn sleep(s int) { C.sleep(s) }
