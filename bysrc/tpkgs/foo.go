package foo

func bar() int {
	//	fmt.Println(time.Now())
	foo()
	go foo()
	return 0
}

func foo() {
	println("foo called")
}

func foo1() string {
	return ""
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
