package main

func node0(no int, c chan int) {
	println("node0", no)
	c <- no
}

func node1(no int, c chan int) {
	println("node1", no)
	<-c
	println("node1 done", no)
}

func main() {
	var c = make(chan int, 5)
	go node0(5, c)
	go node1(8, c)
	sleep(5)
}
