package main

type T struct {
	Left, Right *T
}

// type M map[string]M

func main() {
	var t T
	println(t.Left, t.Right)
}
