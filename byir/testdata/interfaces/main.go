package main

type concreteFoo int

func (f *concreteFoo) Foo() {
	println(f)
}

func main() {
	var x concreteFoo = 2
	f(&x)
}

func f(fooer interface{ Foo() }) {
	fooer.Foo()
}
