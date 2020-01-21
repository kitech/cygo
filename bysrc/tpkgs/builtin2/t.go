package main

func main() {
	sz := /*builtin.*/ sizeof(int)
	if false {
		/*builtin.*/ assert(1 == 2)
	}
	sz2 := alignof(int)
	println(sz, sz2)
}
