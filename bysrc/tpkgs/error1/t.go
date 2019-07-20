package main

import "errors"

func main() {
	var v = 5
	println(v)
	var err error
	err = errors.New("hehehe")
	println(err.Error())
}
