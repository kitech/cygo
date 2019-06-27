package main

func main() {
	var v = "abc"
	println(v)
	v = v + "def"
	println(v)
	var slen = len(v)
	println(slen, len(v))

	v1 := v[:3]
	println(len(v1))

	v2 := v[2:]
	println(len(v2))

	v3 := v[2:3]
	println(len(v3))

	v4 := v[3:3]
	println(len(v4))

	// v5 := v[3:2]
	// println(len(v5))

	b1 := v1 == v2
	println(b1)

	b2 := v1 != v2
	println(b2)

	b3 := !b2
	println(b3)
}
