package main

func main() {
	v := []int{1, 2, 3, 4, 5}
	println(v)
	println(len(v))
	// sleep(5)
	for idx, elem := range v {
		println(idx, elem)
	}

	{
		v := []string{"abc", "def", "ghi"}
		println(v)
		for idx, elem := range v {
			println(idx, elem)
		}
	}
}
