package main

import "cxrt/bysrc/tpkgs/lib1"

func main() {
	var v = 5
	println(v)

	lib1.Keep()
}
