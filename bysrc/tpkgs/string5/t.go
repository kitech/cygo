package main

import "unsafe"

type Builder struct {
	// addr *Builder // of receiver, to detect copies by value
	buf []byte
}

// String returns the accumulated string.
func (b *Builder) String() string {
	p := unsafe.Pointer(&b.buf)
	sp := (*string)(p)
	s := *sp
	// return *(*string)(unsafe.Pointer(&b.buf))
	return s
}
func (b *Builder) String2() string {
	return *(*string)(unsafe.Pointer(&b.buf))
}

func main() {
	var v = 5
	println(v)
}
