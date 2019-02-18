package main

import "strconv"

func main() {
	var y float64
	for i := 0; i < 100000000; i++ {
		x, _ := strconv.ParseFloat("2.34567", 64)
		y += x
		x, _ = strconv.ParseFloat("12.34567", 64)
		y += x
		x, _ = strconv.ParseFloat("1.34567", 64)
		y += x
		x, _ = strconv.ParseFloat("12.4567", 64)
		y += x
	}

	println("hello world", int(y))
}
