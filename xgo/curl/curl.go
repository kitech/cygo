package curl

/*
#include <curl/curl.h>
#include <curl/easy.h>
#include <cxrtbase.h>
*/
import "C"

type Curl struct {
	cobj voidptr // unsafe.Pointer
}

var inited = false

func init() {
	if inited {
		return
	}
	inited = true

	C.curl_global_init(0)
}
func Version() string {
	cstr := C.curl_version()
	return C.GoString(cstr)
}

func New() *Curl {
	cuh := &Curl{}
	cobj := C.curl_easy_init()
	cuh.cobj = cobj

	C.cxrt_set_finalizer(cuh, curlobj_finalizer)
	return cuh
}

func curlobj_finalizer(ptr voidptr /*unsafe.Pointer*/) {
	cuh := (*Curl)(ptr)
	cuh.cleanup()
}

type Slist struct {
	Cobj voidptr // unsafe.Pointer
}

func NewSlist() *Slist {
	lst := &Slist{}
	C.cxrt_set_finalizer(lst, slist_finalizer)
	return lst
}

func slist_finalizer(ptr voidptr /*unsafe.Pointer*/) {
	lst := (*Slist)(ptr)
	if lst == nil {
		return
	}
	lst.free()
}

func (lst *Slist) Append(line string) {
	clst := C.curl_slist_append(lst.Cobj, line)
	lst.Cobj = clst
}
func (lst *Slist) free() {
	C.curl_slist_free_all(lst.Cobj)
}

func (ch *Curl) cleanup() {
	C.curl_easy_cleanup(ch.cobj)
}
func (ch *Curl) perform() {
	C.curl_easy_perform(ch.cobj)
}

func (ch *Curl) setopt(opt int, val voidptr /*unsafe.Pointer*/) int {
	rv := C.curl_easy_setopt(ch.cobj, opt, 2)
	return rv
}
func (ch *Curl) Getinfo(opt int, val voidptr /*unsafe.Pointer*/) int {
	rv := C.curl_easy_getinfo(ch.cobj, opt, val)
	return rv
}

func (ch *Curl) Get() {

}
func (ch *Curl) Post() {

}
func (ch *Curl) Put() {

}
func (ch *Curl) Delete() {

}
func (ch *Curl) Options() {

}
func (ch *Curl) Propfind() {

}

const (
	OK = C.CURLE_OK
)

var (
	OK2 = C.CURLE_OK
)

func init() {

}
func init() {

}
func Keep() {}
