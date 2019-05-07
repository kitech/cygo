package m

import "github.com/pwaller/go2ll/testdata/m/a"

type F struct{ x int }

func (f *F) set(x int) {
	f.x = x
}

func M() {
	a.A()
	var f F
	f.set(1)
	println(f.x)
}
