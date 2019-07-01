package main

func main() {
	var v = 5
	println(v)

	v2 := []int{1, 2, 3}
	v2 = append(v2, 4)
	println(v2)
	println(len(v2), 4)

	v2 = append(v2, 5, 6, 7, 8)
	println(len(v2), 8)

}
