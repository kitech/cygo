package main

const (
	mutexLocked = 1 << iota // mutex is locked
	mutexWoken
	mutexStarving
	mutexWaiterShift = iota
	mutexWaiterShift2
	mutexWaiterShift3
)

const (
	gs1 = "abc"
	gs2 = "efg"
)

var g1 = 1
var b1 = true

func main() {
	var v = 5
	println(v)
}
