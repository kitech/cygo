package main

import "github.com/pwaller/go2ll/testdata/sha1/sha1"

func main() {
	d := sha1.New()
	d.Write([]byte("Hello world"))
	println(string(d.Sum(nil)))
}