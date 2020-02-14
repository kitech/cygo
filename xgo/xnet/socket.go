package xnet

/*
#include <errno.h>
#include <string.h>
#include <sys/socket.h>
#include <netinet/in.h>
#include <arpa/inet.h>
#include <netdb.h>
#include <fcntl.h>
*/
import "C"
import (
	"xgo/xstrconv"
	"xgo/xstrings"
)

func Keep() {}

type Socket struct {
	fd   int
	eno  int
	emsg string
}

func NewSocket() *Socket {
	fd := C.socket(C.AF_INET, C.SOCK_STREAM, 0)
	fd2 := int(fd)
	sock := &Socket{}
	sock.fd = fd2
	//
	return sock
}

func (sk *Socket) Connect(address string, port int) error {
	var sa = &C.struct_sockaddr_in{}
	sa.sin_family = C.AF_INET
	sa.sin_port = C.htons(port)
	C.inet_pton(C.AF_INET, address.ptr, &sa.sin_addr.s_addr)
	var sa4sz = &C.struct_sockaddr_in{}
	var rv = C.connect(sk.fd, sa, sizeof(*sa4sz))
	if rv != 0 {
	}
	println(sk.fd, sa.sin_port)
	return nil
}

func (sk *Socket) Close() error {
	C.close(sk.fd)
	return nil
}

func (sk *Socket) Bind(port int) error {
	var sa = &C.struct_sockaddr_in{}
	sa.sin_family = C.AF_INET
	sa.sin_port = C.htons(port)
	// rv := C.bind(sk.fd, sa, sizeof(C.struct_sockaddr_in(0))) // TODO compiler
	var sa4sz = &C.struct_sockaddr_in{}
	rv := C.bind(sk.fd, sa, sizeof(*sa4sz))
	if rv != 0 {
	}
	return nil
}

func (sk *Socket) Listen() error {
	rv := C.listen(sk.fd, 128)
	if rv != 0 {
		println("listen error", rv)
	}
	return nil
}

func (sk *Socket) Accept() error {
	var sa = &C.struct_sockaddr_in{}
	sa.sin_family = C.AF_INET
	var sa4sz = &C.struct_sockaddr_in{}
	rv := C.accept(sk.fd, sa, sizeof(*sa4sz))
	if rv < 0 {
		println("accept error", rv)
	}
	return nil
}

func (sk *Socket) Read(b []byte) error {
	return nil
}

func (sk *Socket) Write(b []byte) error {
	return nil
}

func Dial(address string) (*Socket, error) {
	arr := xstrings.Split(address, ":")
	port := xstrconv.Atoi(arr[1])
	sk := NewSocket()
	err := sk.Connect(arr[0], port)
	if err != nil {
		return nil, err
	}
	return sk, nil
}

///
type SockAddr struct {
	family u16
	port   u16
	addr   u32
	zero   [8]byte
}

type AddrInfo struct {
	ai_flags     int
	ai_family    int
	ai_socktype  int
	ai_protocol  int
	ai_addrlen   C.socklen_t
	ai_addr      *SockAddr
	ai_canonname byteptr
	ai_next      *AddrInfo
}

func (ai *AddrInfo) clone() *AddrInfo {
	newai := &AddrInfo{}
	*newai = *ai
	newai.ai_next = nil
	if ai.ai_canonname != nil {
		s := gostring(ai.ai_canonname)
		newai.ai_canonname = memdup3(ai.ai_canonname, s.len)
	}
	if ai.ai_addr != nil {
		v := &SockAddr{}
		*v = *(ai.ai_addr)
		newai.ai_addr = v
	}
	return newai
}

func Lookup(host string) *AddrInfo {
	var hires *AddrInfo
	rv := C.getaddrinfo(host.cstr(), 0, 0, &hires)
	if rv != 0 {
		println(cerrmsg())
		return nil
	}
	if hires != nil {
		// defer C.freeaddrinfo(hires)
	}

	var newhires *AddrInfo
	curai := hires
	for curai != nil {
		newai := curai.clone()
		newai.ai_next = newhires
		newhires = newai
		curai = curai.ai_next
	}
	return newhires
}

func (info *AddrInfo) getip() string {
	// ipstr := [128]byte
	ipstr := malloc3(128)
	ipstr2 := C.inet_ntop(info.ai_family, &info.ai_addr.addr, ipstr, 128)
	iptmp := gostring_clone(ipstr2)
	return iptmp
}

func cfcntl(fd int, opt int, val int) int {
	flagsx := C.fcntl(fd, C.F_GETFL, 0)
	return flagsx
}

func fd_set_nonblocking(fd int, isNonBlocking bool) bool {
	flags := cfcntl(fd, C.F_GETFL, 0)
	var onb int = C.O_NONBLOCK
	oldv := flags & onb
	old := oldv > 0
	if isNonBlocking == old {
		return old
	}

	var newflags int
	if isNonBlocking {
		newflags = flags | onb
	} else {
		// newflags = flags & ~C.O_NONBLOCK
	}
	rv := C.fcntl(fd, C.F_SETFL, newflags)
	return rv == 0
}

func fd_set_reuseaddr(fd int) int {
	val := 1
	rv := C.setsockopt(fd, C.SOL_SOCKET, C.SO_REUSEADDR, &val, sizeof(int(0)))
	if rv != 0 {
		// vpp.prtcerr('xnreuseaddr $rv')
		prtcerr("xnreuseaddr")
	}
	return rv
}

/*
pub fn lookup(host string) ?&AddrInfo {
	hires := (*AddrInfo)(0)
	rv := C.getaddrinfo(host.str, 0, 0, &hires)
	if rv != 0 {
		return error(vpp.cerrmsg())
	}
	defer {
		if !isnil(voidptr(hires)) {
			C.freeaddrinfo(hires)
		}
	}
	// clone one and free C scope allocated one
	mut newhires := (*AddrInfo)(0)
	mut curai := hires
	for !isnil(voidptr(curai)) {
		mut newai := curai.clone()
		newai.ai_next = newhires
		newhires = newai
		curai = curai.ai_next
	}
	return newhires
}

pub fn (info &AddrInfo) getip() string {
	ipstr := [128]byte
	ipstr2 := C.inet_ntop(info.ai_family, &info.ai_addr.addr, &ipstr[0], 128)
	iptmp := tos_clone(ipstr2)
	return iptmp
}

pub fn fd_set_nonblocking(fd int, isNonBlocking bool) bool {
    flags := C.fcntl(fd, C.F_GETFL, 0)
    old := (flags & C.O_NONBLOCK) > 0
    if isNonBlocking == old { return old }

	newflags := if isNonBlocking { flags | C.O_NONBLOCK } else { flags & ~C.O_NONBLOCK }
    rv := C.fcntl(fd, C.F_SETFL, newflags)
    return rv == 0
}

fn fd_set_reuseaddr(fd int) int {
	val := 1
	rv := C.setsockopt(fd, C.SOL_SOCKET, C.SO_REUSEADDR, &val, sizeof(int))
	if rv != 0 { vpp.prtcerr('xnreuseaddr $rv') }
	return rv
}

pub fn isip4(ip string) bool {
	fields := ip.split('.')
	if fields.len != 4 { return false }
	for field in fields {
		if !vpp.isdigit(field){
			return false
		}
		n := strconv.atoi(field)
		if n < 0 || n > 255 {
			return false
		}
	}
	return true
}

pub fn splithost(hostport string) (string,string) {
	arr := hostport.split(':')
	if arr.len == 2 {
		return arr[0],arr[1]
	}
	return arr[0], '0'
}

*/
