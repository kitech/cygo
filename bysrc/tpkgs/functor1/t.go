package main

func foo1(f1 func()) {

}
func foo2(f1 func(int)) {

}
func foo3() func() {
	return nil
}
func foo4() func(int) {
	return nil
}
func foo5() func(int) string {
	return nil
}

/*
func foo6() func(int) (string, float32) {
	return nil
}
*/

type bar1 struct {
	f1 func()
	f2 func(int)
	f3 func(string)
	f4 func() int
	f5 func() []string
}

func main() {
	var v = 5
	println(v)

	b1 := &bar1{}
	// b1.f1() // TODO compiler
	f1 := b1.f1
	f1()
	f2 := b1.f2
	f2(0)
}
