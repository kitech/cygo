package main

func routine1(sleepsec int) {
	for {
		println("", sleepsec, 0)
		sleep(sleepsec)
		break
	}
}

func main() {
	routine1(3)
}
