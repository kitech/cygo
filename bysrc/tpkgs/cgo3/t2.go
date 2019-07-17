package main

/*
#include <time.h>
*/
import "C"

func mytime() {
	tm := C.time(nil)
}
