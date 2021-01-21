package main

/*
#cgo LDFLAGS: -lcurl

#include <sys/socket.h>
#include <unistd.h>
#include <errno.h>

*/
import "C"

/*
hehehehhe
*/

// aaaaaa
func main() {
	var v = 5
	println(v)
	C.sleep(1)
	var fd int = int(C.socket(C.AF_INET, C.SOCK_STREAM, 0))
	println(fd)
	// println(C.errno)
	var pid int = int(C.getpid())
	println(pid)

	t2foo0() // from ./t2.go
}
