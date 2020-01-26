package builtin

// don't use other packages, only C is supported

/*
#include <stdlib.h>
#include <stdio.h>
*/
import "C"

func keep() {}

// func panic()   {}
func panicln() {}
func fatal()   {}
func fatalln() {}

func malloc3(sz int) voidptr {
	ptr := C.cxmalloc(sz)
	return ptr
}
func realloc3(ptr voidptr, sz int) voidptr {
	ptr2 := C.cxrealloc(ptr, sz)
	return ptr2
}
func free3(ptr voidptr) {
	C.cxfree(ptr)
}

//[nomangle]
func assert()
func sizeof() int
func alignof() int
func offsetof() int

//export hehe_exped
func hehe(a int, b string) int {
	return 0
}

type mirstring struct {
	ptr voidptr
	len int
}

func (s string) split(sep string) []string {
	return nil
}
func (s string) trimsp() string {
	return s
}

func (i int) repr() string {
	return ""
}

func (i float64) repr() string {
	return ""
}
func (i float32) repr() string {
	return ""
}
