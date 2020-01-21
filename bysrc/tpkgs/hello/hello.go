package main

func main() {
	go hello0()
}

func hello0() {

}

func hello1() {
	c := make(chan int, 3)
	c <- 1
}

type atree struct {
	a int
	b string
}

func hello2() {
	// v := &atree{}
	// v1 := atree{}
	// println(v, v1)
}

var gv123 int = 1

func init() {
	gv123 = 2
}

func init() {
	gv123 = 3
}
