package main

import "strconv"

func main() {
	var y float64
	for i := 0; i < 200000000; i++ {
		x, _ := strconv.ParseFloat("12.34567", 64)
		y += x
		z, _ := strconv.ParseFloat("2.34567", 64)
		y += z
	}

	println("hello world", int(y))
}
