package main

func main() {
	var v = 5
	println(v)

	func() {}()
	func(a int, b bool) {}(1, 2)
	f3 := func() bool { return true }()
}
