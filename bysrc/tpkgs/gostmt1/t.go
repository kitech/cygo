package main

func bar() int {
	sleep(1)
	//	fmt.Println(time.Now())
	foo()
	go foo()
	go foo2(1, 2)
	return 0
}

func foo() {
	println("foo called")
}

func foo1() string {
	return ""
}

func foo2(a int, b int) {
	println("foo2 called", a)
}

func main() {

	println(5)

	println("aaa", 123, cxgettid())

	bar()
	sleep(5)
}
