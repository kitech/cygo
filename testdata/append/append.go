package main

func main() {
	var x []int
	println(x)
	x = append(x, 1)
	println(x, x[0])
	x = append(x, 2)
	println(x, x[0], x[1])
	x = append(x, 123456)
	println(x, x[0], x[1], x[3])

	x = append(x, 99, 98, 87)

	dumpSlice(x)
}

func dumpSlice(x []int) {
	for i, v := range x {
		println(i, v)
	}
}
