package main

import "unsafe"

func main() {
	var b byte
	bp1 := add1(&b)
	println(&b, bp1)
	// from
}

func add1(p *byte) *byte {
	return (*byte)(unsafe.Pointer(uintptr(unsafe.Pointer(p)) + 1))
}

func fromFutexWakeup() {
	*(*int32)(unsafe.Pointer(uintptr(0x1006))) = 0x1006
}

func abort()
