package main

/*
#cgo LDFLAGS: -lhehe1

#include <sys/socket.h>
#include <unistd.h>

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
	var fd int = C.socket(C.AF_INET, C.SOCK_STREAM, 0)
	println(fd)
	println(C.errno)
	var pid int = C.getpid()
	println(pid)

	t2foo0()
}
