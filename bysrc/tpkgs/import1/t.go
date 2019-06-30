package main

import "cxrt/bysrc/tpkgs/lib1"
import "cxrt/bysrc/tpkgs/lib2"
import lib3a "cxrt/bysrc/tpkgs/lib3"

func main() {
	var v = 5
	println(v)

	lib1.Keep()
	lib2.Keep()
	lib3a.Keep()
}
