package xnet

/*
#include <errno.h>
#include <string.h>
#include <sys/socket.h>
#include <netinet/in.h>
#include <arpa/inet.h>
*/
import "C"
import (
	"gopp/xstrconv"
	"gopp/xstrings"
)

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
	var rv = C.connect(sk.fd, sa, sizeof(C.struct_sockaddr_in))
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
	rv := C.bind(sk.fd, sa, sizeof(C.struct_sockaddr_in))
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
	rv := C.accept(sk.fd, sa, sizeof(C.struct_sockaddr_in))
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
