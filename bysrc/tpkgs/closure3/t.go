package main

func foo() {

}

type bar struct {
	b1 int
	b2 bool
}

func main() {
	var v = 5
	println(v)

	s1 := "hehehe"
	fv1 := 1.23
	stv1 := &bar{}

	f1 := func() {
		println(v)
		println(s1)
		println(fv1)
		println(stv1)

		cv1 := false
		println(cv1)
	}

	f1()

	// f2 := foo
	// f2()
}
