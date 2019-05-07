package main

func main() {
	val := true
	f := func(b bool) bool {
		println("f():", b)
		return b && val
	}
	val = false
	if f(true) {
		println("main(): true")
	} else {
		println("main(): false")
	}

	makeFunc(true)()
	makeFunc(false)()
}

func g() {
	println("g()")
}

func makeFunc(cond bool) func() {
	var x int
	f := func() {
		println("f() closure:", x)
	}
	x = 42

	var ret func()
	if cond {
		ret = f
	} else {
		ret = g
	}
	return ret
}
