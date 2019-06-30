package main

func hasSuffix(s string, sfx string) bool {
	start := len(s) - len(sfx)
	if len(s) >= len(sfx) && s[start:] == sfx {
		return true
	}
	return false
}

func main() {
	s := "abcdefg"
	s1 := s[4:]
	println(s1)

	has1 := hasSuffix(s, "efg")
	println(has1)

	has2 := hasSuffix(s, "hhe")
	println(has2)

	has3 := hasSuffix(s, "")
	println(has3)
}
