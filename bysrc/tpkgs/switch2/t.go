package main

func main() {
	var v = 5
	println(v)
	var v2 = 3

	switch {
	case v > 1, v > 2:
		v2 = 5
		v2 = 6
	case v < 1, v < 0:
		v2 = 7
		v2 = 8
		// fallthrough // TODO
	case v == 123:
		v2 = 123

	}
	println(v2)
}
