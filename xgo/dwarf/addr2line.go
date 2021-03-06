package dwarf

/*
#cgo LDFLAGS: -ldwarf

#include <execinfo.h>
#include <libdwarf/dwarf.h>
#include <libdwarf/libdwarf.h>
#include <libelf.h>

// for type resolve
extern int init_elf_dwarf2();
extern void rtdebug2_addr2line(void*, char*, int*);
*/
import "C"

func init() {
	if false {
		C.init_elf_dwarf2()
	}
}

func addr2line1(addr voidptr) (string, int) {
	buf := C.cxmalloc(100)
	lineno := 0
	C.rtdebug2_addr2line(addr, buf, &lineno)
	filex := gostring(buf)
	return filex, lineno
}

func test_addr2line() {
	println("in here 111")
	buf := C.cxmalloc(100)
	var addr voidptr = &test_addr2line
	lineno := 0
	C.rtdebug2_addr2line(addr, buf, &lineno)
	filex := gostring(buf)
	var file string = filex
	println(lineno, file)
}
