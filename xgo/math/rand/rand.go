package rand

/*
#include <stdlib.h>
#include <time.h>
*/
import "C"

func dummying() {
	var p voidptr
}

func init() {
	C.srand(C.time(0))
}

func Int() int {
	return C.rand()
}

func Intn(n int) int {
	rv := C.rand()
	return rv % n
}
