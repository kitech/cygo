package main

func foo1() (int, string) {
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
}
