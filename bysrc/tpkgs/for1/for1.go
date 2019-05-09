package main

func routine1(sleepsec int) {
	for i := 0; i < 5; i++ {
		println("", sleepsec, i)
		sleep(sleepsec)
	}
}

func main() {
	routine1(3)
}
