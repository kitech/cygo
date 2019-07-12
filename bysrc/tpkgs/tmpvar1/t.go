package main

func abc(nums []int) int {
	return 123
}

func efg(nums map[int]int) {

}

func main() {
	var v = 5
	println(v)
	v2 := abc([]int{1, 2, 3})
	efg(map[int]int{1: 1, 2: 2, 3: 3})
}
