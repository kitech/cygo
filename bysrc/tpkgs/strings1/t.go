package main

import "congo/strings"

func main() {
	var v = 5
	println(v)

	s1 := " abc"
	s2 := strings.TrimSpace(s1)
	println(s2)

	s3 := "fOo123"
	println(strings.HasPrefix(s3, "foo"))
	println(strings.HasPrefix(s3, "foo3"))
	println(strings.HasSuffix(s3, "123"))
	println(strings.HasSuffix(s3, "a123"))

	println(strings.Title(s3))
	println(strings.Upper(s3))
	println(strings.Lower(s3))

	println(strings.Contains(s3, "foo"))
	println(strings.Contains(s3, "fOo"))
	println(strings.Contains(s3, "o123"))
}
