package main

func main() {
	bytes()
	strings()
	structs()
}

func bytes() {
	b := []byte("Hello world\x00")
	c := b[6:]
	println(string(c))
}

func strings() {
	s := "hello world"
	println(s[:5])
	println(s[6:])
	println(s[3:5])
}

func structs() {
	type foo struct{ x int }
	s := []foo{{1}, {2}}
	println(s[1:][0].x)
}
