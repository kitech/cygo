package xmath

/*
#include <math.h>
*/
import "C"

func Keep() {}

func Absint(j int) int {
	return C.abs(j)
}

func Absi64(j int64) int64 {
	return C.llabs(j)
}
