package main

func main() {
	bytes()
	strings()
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
