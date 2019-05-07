package main

import "strconv"

func main() {
	bytes()
	strings()
	structs()
	arrays()
	makeslice()
	stringer()
}

func bytes() {
	b := []byte("Hello world\x00")
	c := b[6:]
	println(string(c))
}

func strings() {
	s := "hello world"
	println(s[:5])
	println(s[6:])
	println(s[3:5])
}

func structs() {
	type foo struct{ x int }
	s := []foo{{1}, {2}}
	println(s[1:][0].x)
}

func arrays() {
	x := [...]int{1, 2, 3, 4}
	println(x[:][0], x[:][1], x[:][2], x[:][3])
	println(x[1:][0], x[1:][1], x[1:][2])
	println(x[1:3][0], x[1:3][1])
	println(x[:2][0], x[:2][1])
}

var s []string

func makeslice() {
	n := int32(10)
	s = make([]string, n)
	for i := int32(0); i < n; i++ {
		s[i] = "foo"
	}
}

func stringer() {
	for i := range _AtomicOp_index {
		println(AtomicOp(i + 1).String())
	}
}

type AtomicOp int

const _AtomicOp_name = "addandmaxminnandorsubumaxuminxchgxor"

var _AtomicOp_index = [...]uint8{0, 3, 6, 9, 12, 16, 18, 21, 25, 29, 33, 36}

func (i AtomicOp) String() string {
	i -= 1
	if i >= AtomicOp(len(_AtomicOp_index)-1) {
		return "AtomicOp(" + strconv.FormatInt(int64(i+1), 10) + ")"
	}
	return _AtomicOp_name[_AtomicOp_index[i]:_AtomicOp_index[i+1]]
}
