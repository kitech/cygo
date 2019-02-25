package main

import (
	"github.com/pwaller/go2ll/testdata/sha1/sha1"
)

func main() {
	d := sha1.New()
	input := []byte("Hello world")
	for i := 0; i < 100000000; i++ {
		d.Write(input)
	}
	println(encodeToString(d.Sum(nil)))
}

func encode(dst, src []byte) int {
	for i, v := range src {
		dst[i*2] = hextable[v>>4]
		dst[i*2+1] = hextable[v&0x0f]
	}

	return len(src) * 2
}

const hextable = "0123456789abcdef"

func encodeToString(src []byte) string {
	dst := make([]byte, encodedLen(len(src)))
	encode(dst, src)
	return string(dst)
}

func encodedLen(n int) int { return n * 2 }
