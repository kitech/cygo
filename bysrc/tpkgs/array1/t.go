package main

func main() {
	var v = []int{1, 2, 3}
	println(v)

	var n1 = len(v)
	println(n1)

	var n2 = cap(v)
	println(n2)

	v2 := v[1:2]
	println(v2)

	n3 := len(v2)
	println(n3)

	v[1] = 5

	e1 := v[1]
	println(e1)

	v = nil
}
