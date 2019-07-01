package main

/*
 */
import "C"
import "unsafe"

type sttype struct {
	f1 int
}

type abtype1 = int

type abtype2 int

type abtype3 *int

type mypointer = unsafe.Pointer

func mynewtcm() mypointer

func main() {
	var v = 5
	println(v)
	var p1 unsafe.Pointer
	println(p1)

	sz := unsafe.Sizeof(v)
	println(sz)

	vptr := unsafe.Pointer(&v)
	println(vptr)

	// var vx mypointer = mynewtcm()
	// println(vx)

	// var ptr2 pointer
}
