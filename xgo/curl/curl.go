package curl

/*
#include <curl/curl.h>
#include <curl/easy.h>
#include <cxrtbase.h>
*/
import "C"

type Curl struct {
	cobj        voidptr // unsafe.Pointer
	verbose_    bool
	user_agent_ string
	headers_    map[string]string
	timeoutms_  int
	url_        string
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
func (ch *Curl) perform() int {
	rv := C.curl_easy_perform(ch.cobj)
	return rv
}

func (ch *Curl) setopt(opt int, val voidptr /*unsafe.Pointer*/) int {
	rv := C.curl_easy_setopt(ch.cobj, opt, 2)
	return rv
}
func (ch *Curl) Getinfo(opt int, val voidptr /*unsafe.Pointer*/) int {
	rv := C.curl_easy_getinfo(ch.cobj, opt, val)
	return rv
}

func (ch *Curl) seturl(u string) *Curl {
	ch.url_ = u
	return ch
}
func (ch *Curl) verbose(v bool) *Curl {
	ch.verbose_ = v
	return ch
}
func (ch *Curl) timeoutms(ms int) *Curl {
	ch.timeoutms_ = ms
	return ch
}
func (ch *Curl) user_agent(ua string) *Curl {
	ch.user_agent_ = ua
	return ch
}
func (ch *Curl) header_line(line string) *Curl {
	fields := line.split(":")
	// ch.headers_[fields[0]] = fields[1] // TODO compiler
	return ch
}
func (ch *Curl) headers(hdrs map[string]string) *Curl {
	for k, v := range hdrs {

	}
	return ch
}

func (ch *Curl) prepare() {
	ch.setopt(C.CURLOPT_URL, ch.url_.cstr())
	if ch.verbose_ {
		ch.setopt(C.CURLOPT_VERBOSE, 1)
	}
	if ch.user_agent_ != "" {
		ch.setopt(C.CURLOPT_USERAGENT, ch.user_agent_.cstr())
	}
	if ch.timeoutms_ > 0 {
		ch.setopt(C.CURLOPT_TIMEOUT, ch.timeoutms_)
	}
}

func (ch *Curl) prepmethod(method string) {
	mth := method.tolower()
	switch mth {
	case "get": // default
	case "post":
		ch.setopt(C.CURLOPT_POST, 1)
	case "put":
		ch.setopt(C.CURLOPT_PUT, 1)
	default:
		ch.setopt(C.CURLOPT_CUSTOMREQUEST, method.cstr())
	}
}
func (ch *Curl) do(method string) int {
	ch.prepare()
	ch.prepmethod(method)
	rv := ch.perform()
	return rv
}

func (ch *Curl) Get() {
	ch.do(GET)
}
func (ch *Curl) Post() {
	ch.do(POST)
}
func (ch *Curl) Put() {
	ch.do(PUT)
}
func (ch *Curl) Delete() {
	ch.do(DELETE)
}
func (ch *Curl) Options() {
	ch.do(OPTIONS)
}
func (ch *Curl) Propfind() {
	ch.do(PROPFIND)
}

func Get(u string) {
	ch := New()
	ch.seturl(u)
	ch.Get()
}

const (
	GET      = "GET"
	POST     = "POST"
	PUT      = "PUT"
	DELETE   = "DELETE"
	OPTIONS  = "OPTIONS"
	PROPFIND = "PROPFIND"
)

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
