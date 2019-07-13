package main

type foo struct {
	x int
	y int
}

func main() {
	var v = 5
	println(v)

	var c1 chan *foo
	if false {
		v1 := <-c1
	}
	if false {
		c1 <- nil
	}
}
