package main

/*
#include <time.h>
*/
import "C"

func mytime() C.time_t {
	tm := C.time(nil)
	return tm
}
