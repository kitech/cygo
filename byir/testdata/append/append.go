package main

func main() {
	ints()
	structs()
}

func ints() {
	var x []int
	println(x)
	x = append(x, 1)
	println(x, x[0])
	x = append(x, 2)
	println(x, x[0], x[1])
	x = append(x, 123456)
	println(x, x[0], x[1], x[2])

	x = append(x, 99, 98, 87)

	intsDumpSlice(x)
}

func structs() {
	type foo struct {
		x, y int
	}
	var s []foo
	s = append(s, foo{1, 2})
	s = append(s, foo{3, 4})
	println(s[0].x, s[0].y)
	println(s[1].x, s[1].y)
}

func intsDumpSlice(x []int) {
	for i, v := range x {
		println(i, v)
	}
}
