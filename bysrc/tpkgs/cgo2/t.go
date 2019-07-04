package main

/*
 */
import "C"

type foost struct {
	a int
	b bool
}

type cint = C.int
type cuint = C.uint

type cint1 C.int
type cuint1 C.uint

func main() {
	var v = 5
	println(v)

	i1 := cint(0)

}
