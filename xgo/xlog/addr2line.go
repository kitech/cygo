package xlog

/*
#cgo LDFLAGS: -ldwarf

#include <execinfo.h>
#include <libdwarf/dwarf.h>
#include <libdwarf/libdwarf.h>
#include <libelf.h>

*/
import "C"

func init() {
	C.init_elf_dwarf2()
}

func addr2line1(addr voidptr) (string, int) {
	buf := C.cxmalloc(100)
	lineno := 0
	C.rtdebug2_addr2line(addr, buf, &lineno)
	filex := C.GoString(buf)
	return filex, lineno
}

func test_addr2line() {
	println("in here 111")
	buf := C.cxmalloc(100)
	var addr voidptr = &test_addr2line
	lineno := 0
	C.rtdebug2_addr2line(addr, buf, &lineno)
	filex := C.GoString(buf)
	var file string = filex
	println(lineno, file)
}
