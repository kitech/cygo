package main

import "unsafe"

func foo() unsafe.Pointer {
	return nil
}

func foo2(ptr unsafe.Pointer) unsafe.Pointer {
	return ptr
}

func foo3(ptr unsafe.Pointer) unsafe.Pointer {
	ptr = (uintptr)(5)
	return ptr
}

func main() {
	var v = 5
	println(v)
	unsafe.Sizeof(v)
	unsafe.Alignof(v)
	// unsafe.Offsetof(v)
}
