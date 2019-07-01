package main

type foo struct {
	f1 int
	f2 string
	f3 []int16
}

func main() {
	var v = 5
	println(v)

	var fa foo
	println(fa)
	var fb *foo
	println(fb)
}
