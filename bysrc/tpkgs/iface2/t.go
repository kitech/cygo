package main

type foo1 interface {
	Name(bool) string
	Age() int
}

type foo2 interface {
	Weight(int) int
	Sex() int
}

type foo3 interface {
	country() string
	city(float32) string
	m1()
}

func main() {
	var v = 5
	println(v)
}
