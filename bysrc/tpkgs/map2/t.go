package main

func main() {
	var m1 = map[string]int{"abc": 1, "efg": 2}
	m1["k1"] = 3

	var k2 = "k2"
	m1[k2] = 4

	println(m1)

	// mc := len(m1)
	// println(mc)
}
