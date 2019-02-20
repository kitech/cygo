package main

func main() {
	v := 1
	switch v {
	case 1:
		type u struct{ i int64 }
		var vu u
		println(vu.i)
	case 2:
		type u struct{ i float64 }
		var vu u
		println(vu.i)
	}
}
