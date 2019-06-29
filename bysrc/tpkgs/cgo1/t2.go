package main

/*
#include <stdlib.h>

void t2foo0_() {
}
*/
import "C"

func t2foo0() {
	C.t2foo0_()
}
