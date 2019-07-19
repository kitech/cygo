package main

func foo() {

}
func foo2() {

}
func foo3() {
}

func main() {
	defer foo()
	var v = 5
	println(v)
	defer foo2()
	if false {
		v = 6
		return
	}
	defer foo3()
	return
}
