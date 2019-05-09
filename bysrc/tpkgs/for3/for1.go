package main

func routine1(sleepsec int) {
	var i = 0
	for i = 0; i < 5; i++ {
		println("", sleepsec, i)
		sleep(sleepsec)
	}
	for i := 0; i < 3; i++ {
		println("", sleepsec, i)
	}
}

func main() {
	routine1(3)
}
