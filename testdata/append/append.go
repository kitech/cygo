package main

func main() {
	// x := []int{1, 2, 3, 4}
	var x []int
	x = append(x, 1)
	x = append(x, 1, 2, 3, 4)
	println(x[0], x[1], x[len(x)-1])
}
