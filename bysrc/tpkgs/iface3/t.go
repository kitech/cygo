package main

type foo1 interface {
	Name(bool) string
	Age() int
}

type foo1impl struct {
}

func (this *foo1impl) Name(a bool) string {
	return "bob"
}
func (this *foo1impl) Age() int {
	return 5
}

func getfoo1() foo1 {
	f1 := &foo1impl{}
	return f1
}

func main() {
	var v = 5
	println(v)

	var f1 foo1
	f1 = getfoo1()
	f1.Age()
	f1.Name(true)
}
