package main

func main() {
	str := "abcdefg"
	for idx, ch := range str {
		println(idx, ch)
	}
	for _, ch := range str {
		println(ch)
	}
}
