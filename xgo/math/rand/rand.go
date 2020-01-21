package rand

/*
#include <stdlib.h>
#include <time.h>
*/
import "C"
import "unsafe"

func dummying() {
	var p unsafe.Pointer
}

func init() {
	C.srand(C.time(0))
}

func Int() int {
	return C.rand()
}
