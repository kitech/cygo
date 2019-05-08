package foo

func bar() int {
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

type color struct {
	r byte
	g byte
	b byte
	a byte
}

func main() {
	c := &color{}
	println(c)

	println(5)

	println("aaa", 123, gettid())

	bar()
	sleep(5)
}
