package main

type foo struct {
	f1 int
	f2 string
	// f3 []int16
}

func main() {
	var v = 5
	println(v)

	f1 := &foo{1, "abc"}

	println(f1.f2)

	f2 := &foo{f2: "efg", f1: 5}
	println(f2.f2)
}
