package main

func main() {
	var v = []string{"abc", "def"}
	println(v)

	ch := v[1][2]
	println(ch)
	if ch != 'f' {
		println("err", ch)
	}

	v[1][2] = 'g'
	ch2 := v[1][2]
	println(ch2)

	v = nil
}
