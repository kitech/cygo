package main

func main() {
	bytes()
	strings()
	structs()
	arrays()
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

func arrays() {
	x := [...]int{1, 2, 3, 4}
	println(x[:][0], x[:][1], x[:][2], x[:][3])
	println(x[1:][0], x[1:][1], x[1:][2])
	println(x[1:3][0], x[1:3][1])
	println(x[:2][0], x[:2][1])
}
