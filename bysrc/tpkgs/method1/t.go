package main

type foo struct {
	a int
}

func (this *foo) bar1() {

}

func (this *foo) bar2() {

}

func (this *foo) bar3() int {
	return 5
}

func (this *foo) bar4(a int) int {
	return 5
}

func bar0() {

}

func main() {
	fo := &foo{}
	println(fo)
	fo.bar1()
	v := fo.bar3()
	println(v)
	fo.bar4(6)
}
