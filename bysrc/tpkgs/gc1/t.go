package main

import "unsafe"

func myfinal(ptr unsafe.Pointer) {
	println(ptr)
}
func dosomealloc(a int) {
	s := "abc"
	for i := 0; i < 9; i++ {
		s = s + "efghijklmn"
		cxrt_set_finalizer(s, myfinal)
		sleep(1)
	}
	println(len(s), a)
}

func main() {
	var v = 5
	println(v)

	for i := 0; i < 12345; i++ {
		go dosomealloc(i)
		sleep(1)
	}
	sleep(5)
}
