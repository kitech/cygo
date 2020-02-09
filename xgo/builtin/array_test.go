package builtin

func test_build () {
	arr := make([]int, 5)
	arr[0] = 111
	arr[1] = 222
	arr[2] = 333
	arr[3] = 444
	a3 := arr[3]
	println(a3)
	alen := arr.len()
	println(alen)

	arr.delete(4)
	alen = arr.len()
	println(alen)

	arr.delete(0)
	alen = arr.len()
	println(alen)
	for idx,e in arr {
		println(idx, e)
	}

	arr.clear()
	alen = arr.len()
	println(alen)
}
