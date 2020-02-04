package curl

/*
#include <curl/curl.h>
#include <curl/easy.h>
#include <cxrtbase.h>

CURL* curl_easy_init();
*/
import "C"

type Curl struct {
	cobj        voidptr // unsafe.Pointer
	verbose_    bool
	user_agent_ string
	headers_    map[string]string
	timeoutms_  int
	url_        string
	connonly_   bool
	uapolicy_   int
	resolves_   map[string]string // domain => ip

	// result
	rcvlen usize
	errbuf voidptr
	rsp    *Response
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
	cuh.rsp = &Response{}

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
func (ch *Curl) uapolicy(uap int) *Curl {
	ch.uapolicy_ = uap
	return ch
}

func (ch *Curl) header_line(line string) *Curl {
	fields := line.split(":")
	k := fields[0]
	v := fields[1]
	ch.headers_[k] = v
	// ch.headers_[fields[0]] = fields[1] // TODO compiler, dead cycle
	return ch
}
func (ch *Curl) headers(hdrs map[string]string) *Curl {
	for k, v := range hdrs {
		ch.headers_[k] = v
	}
	return ch
}
func (ch *Curl) connonly(v bool) *Curl {
	ch.connonly_ = v
	return ch
}
func (ch *Curl) recv() {
	C.curl_easy_recv(ch.cobj, nil, 0, nil)
}
func (ch *Curl) send() {
	C.curl_easy_send(ch.cobj, nil, 0, nil)
}

func (ch *Curl) setopt(opt int, val voidptr /*unsafe.Pointer*/) int {
	rv := C.curl_easy_setopt(ch.cobj, opt, val)
	return rv
}

func header_cltcb(buf voidptr, size usize, nitem usize, cbval voidptr) usize {
	assert(size == 1)
	ch := (*Curl)(cbval)
	rsp := ch.rsp
	// xlog.Println(size, nitem, cbval)
	s := gostringn(buf, nitem)
	// xlog.Println(size, nitem, cbval, s)
	kv := s.split(": ")
	if kv.len() != 2 {
		// first line
		rsp.Stline = s.trimsp()
	} else {
		// rsp.Headers[kv[0]] = kv[1] // TODO compiler
		k := kv[0]
		v := kv[1]
		v = v.trimsp()
		rsp.Headers[k] = v
		// xlog.Println(size, nitem, cbval, k, v, s.trimsp())
	}

	return nitem
}

func send_cltcb(buf voidptr, size usize, nitem usize, cbval voidptr) usize {
	assert(size == 1)
	ch := (*Curl)(cbval)
	rsp := ch.rsp

	return nitem
}
func recv_cltcb(buf voidptr, size usize, nitem usize, cbval voidptr) usize {
	assert(size == 1)
	ch := (*Curl)(cbval)
	rsp := ch.rsp

	ch.rcvlen += nitem
	s := gostringn(buf, nitem)
	rsp.Data += s

	return nitem
}

func debug_cltcb(chobj voidptr, type_ int, data charptr, size usize, cbval voidptr) int {
	switch type_ {
	case C.CURLINFO_TEXT:
	case C.CURLINFO_HEADER_OUT:
	case C.CURLINFO_DATA_OUT:
	case C.CURLINFO_SSL_DATA_OUT:
	case C.CURLINFO_HEADER_IN:
	case C.CURLINFO_DATA_IN:
	case C.CURLINFO_SSL_DATA_IN:
	default:
	}
	return 0
}

func (ch *Curl) setoptfunc(opt int, fnptr voidptr, cbval voidptr) int {
	rv := ch.setopt(opt, fnptr)
	switch opt {
	case C.CURLOPT_READFUNCTION:
		ch.setopt(C.CURLOPT_READDATA, cbval)
	case C.CURLOPT_WRITEFUNCTION:
		ch.setopt(C.CURLOPT_WRITEDATA, cbval)
	case C.CURLOPT_HEADERFUNCTION:
		ch.setopt(C.CURLOPT_HEADERDATA, cbval)
	case C.CURLOPT_DEBUGFUNCTION:
		ch.setopt(C.CURLOPT_DEBUGDATA, cbval)
	case C.CURLOPT_RESOLVER_START_FUNCTION:
		ch.setopt(C.CURLOPT_RESOLVER_START_DATA, cbval)
	default:
		assert(1 == 2)
	}
	return rv
}

func (ch *Curl) prepare() {
	ch.setopt(C.CURLOPT_URL, ch.url_.cstr())
	if ch.verbose_ {
		ch.setopt(C.CURLOPT_VERBOSE, 1)
	}
	if ch.user_agent_ != "" {
		ch.setopt(C.CURLOPT_USERAGENT, ch.user_agent_.cstr())
	} else {
		if ch.uapolicy_ == UAP_HUMAN {
			ua := rand_humanua()
			ch.setopt(C.CURLOPT_USERAGENT, ua.cstr())
		} else if ch.uapolicy_ == UAP_RANDOM {
		}
	}
	if ch.timeoutms_ > 0 {
		ch.setopt(C.CURLOPT_TIMEOUT, ch.timeoutms_)
	}
	if ch.connonly_ {
		ch.setopt(C.CURLOPT_CONNECT_ONLY, 1)
	}
	hdrlst := NewSlist()
	for k, v := range ch.headers_ {
		line := k + ": " + v
		hdrlst.Append(line)
	}
	ch.setopt(C.CURLOPT_HTTPHEADER, hdrlst.Cobj)

	ch.setopt(C.CURLOPT_HEADERFUNCTION, header_cltcb)
	ch.setopt(C.CURLOPT_HEADERDATA, ch)
	ch.setopt(C.CURLOPT_READFUNCTION, recv_cltcb)
	ch.setopt(C.CURLOPT_READDATA, ch)
	if false {
		ch.setopt(C.CURLOPT_WRITEFUNCTION, send_cltcb)
		ch.setopt(C.CURLOPT_WRITEDATA, ch)
	}
	ch.errbuf = malloc3(C.CURL_ERROR_SIZE + 1)
	ch.setopt(C.CURLOPT_ERRORBUFFER, ch.errbuf)
	ch.setopt(C.CURLOPT_DEBUGFUNCTION, debug_cltcb)
	ch.setopt(C.CURLOPT_DEBUGDATA, ch)
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
func (ch *Curl) doreq(method string) int {
	ch.prepare()
	ch.prepmethod(method)
	rv := ch.perform()
	ch.rsp.Ret = rv
	return rv
}

func (ch *Curl) Get() *Response {
	ch.doreq(GET)
	return ch.getresp()
}
func (ch *Curl) Post() *Response {
	ch.doreq(POST)
	return ch.getresp()
}
func (ch *Curl) Put() *Response {
	ch.doreq(PUT)
	return ch.getresp()
}
func (ch *Curl) Delete() *Response {
	ch.doreq(DELETE)
	return ch.getresp()
}
func (ch *Curl) Options() *Response {
	ch.doreq(OPTIONS)
	return ch.getresp()
}
func (ch *Curl) Propfind() *Response {
	ch.doreq(PROPFIND)
	return ch.getresp()
}

func (ch *Curl) Request(method string) *Response {
	ch.doreq(method)
	return ch.getresp()
}

func Get(u string) *Response {
	ch := New()
	ch.seturl(u)
	ch.doreq(GET)
	return ch.getresp()
}

func (ch *Curl) Getinfo(opt int, val voidptr /*unsafe.Pointer*/) int {
	rv := C.curl_easy_getinfo(ch.cobj, opt, val)
	return rv
}
func (ch *Curl) Strerror(code int) string {
	rv := C.curl_easy_strerror(code)
	return gostring(rv)
}

func (ch *Curl) getresp() *Response {
	rsp := ch.rsp
	ch.Getinfo(C.CURLINFO_RESPONSE_CODE, &rsp.Stcode)
	ch.Getinfo(C.CURLINFO_CONTENT_LENGTH_DOWNLOAD_T, &rsp.Cclen)
	if rsp.Cclen == -1 {
		if ch.rcvlen > 0 {
			rsp.Cclen = ch.rcvlen
		}
	}
	rsp.Errmsg = gostring(ch.errbuf)
	return rsp
}

type Request struct {
	Method string
	Requrl string
}

type Response struct {
	Ret     int
	Stcode  int
	Stline  string
	Errmsg  string
	Cclen   int64
	Headers map[string]string
	Data    string

	reqobj *Request
}

// network ok?
func (rsp *Response) Ok() bool { return rsp.Ret == OK }

// protocol ok?
func (rsp *Response) Is10x() bool {
	return rsp.Stcode >= 100 && rsp.Stcode < 200
}
func (rsp *Response) Is20x() bool {
	return rsp.Stcode >= 200 && rsp.Stcode < 300
}
func (rsp *Response) Is30x() bool {
	return rsp.Stcode >= 300 && rsp.Stcode < 400
}
func (rsp *Response) Is40x() bool {
	return rsp.Stcode >= 400 && rsp.Stcode < 500
}
func (rsp *Response) Is50x() bool {
	return rsp.Stcode >= 500 && rsp.Stcode < 600
}

func (rsp *Response) Relocation() string {
	for k, v := range rsp.Headers {
		k1 := k.tolower()
		if k1 == "location" {
			v1 := v
			if v1.prefixed("//") {
				return "https:" + v1
			} else {
				return v1
			}
		}
	}
	return ""
}

func (rsp *Response) Repr1() string {
	str := "ret=" + rsp.Ret.repr() + " "
	str += "stcode=" + rsp.Stcode.repr() + " "
	str += "cclen=" + rsp.Cclen.repr() + " "
	str += "hdrcnt=" + rsp.Headers.len().repr() + " "
	return str
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
	OK    = C.CURLE_OK
	AGAIN = C.CURLE_AGAIN

	// t1 = C.CURLOPT_HTTPPOST
)

var (
	OK2 = C.CURLE_OK
)

func init() {
	if 1 == 1 {
	}
}
func init() {
	if 1 == 2 {
	}
}
func Keep() {}
