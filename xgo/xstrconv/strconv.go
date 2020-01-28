package xstrconv

/*
#include <stdlib.h>
*/
import "C"

func Atoi(s string) int {
	rv := C.atoi(s.ptr)
	return rv
}
func Atol(s string) i64 {
	rv := C.atoll(s.ptr)
	return rv
}
func Atof(s string) f32 {
	rv := C.atof(s.ptr)
	return rv
}
