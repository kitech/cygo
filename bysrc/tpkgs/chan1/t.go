package main

func node0(no int, c chan int) {
	println("node0", no)
	c <- no
}

func node1(no int, c chan int) {
	println("node1", no)
	rno := <-c
	println("node1 done", no, rno)
}

func main() {
	var c = make(chan int, 5)
	go node0(51, c)
	go node1(81, c)
	sleep(5)
}
