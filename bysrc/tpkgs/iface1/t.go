package main

func foo(x interface{}) {
	println(x)
}

func foo2(x interface{}) interface{} {
	var v interface{} = 5
	return v
}

type bar struct {
	f1 int
	f2 string
	f3 interface{}
	f4 interface{}
	f5 interface{}
}

func main() {
	var v = 5
	println(v)

	var vx1 interface{} = 5
	println(vx1)
	pvx1 := vx1.(int)
	println(pvx1)

	var vx2 interface{} = "abc"
	println(vx2)

	var vx3 interface{} = 1.2345
	println(vx3)

	vx4 := foo2(vx3)
	println(vx4)

	b1 := &bar{}
	println(b1)
	b1.f1 = 123
	b1.f2 = "abc"
	b1.f3 = 8
	b1.f4 = vx4

	b1.f5 = []string{"abc", "efg"}
}
