package main

func foo1() (iv int, sv string) {
	if true {
		iv = 3
	}

	if false {
		return
	}

	return 5, "abc"
}

func foo2() (int, error) {
	var err error
	return 5, err
}

func main() {
	var v = 5
	println(v)

	v1, s1 := foo1()
	println(v1, s1)
}
