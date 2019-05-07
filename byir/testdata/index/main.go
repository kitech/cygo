package main

func main() {
	for i, v := range [...]int{42, 2, 3, 4, 5} {
		println(i, v)
	}
}
