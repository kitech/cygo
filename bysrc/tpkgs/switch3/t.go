package main

const (
	C1 = 1
	C2 = 2
	C3 = 3
	C4 = 4
)

func main() {
	var v = 5
	var v2 = 6
	switch i := 0; v {
	case C1:
		switch v2 {
		case C1:
		case C2:
			break
		default:
		}
		println(1)
	case C2:
		if 1 == 1 {
			break
		}
		println(2)
	case C3, C4:
		println(3, 4)
		if 1 == 2 {
			fallthrough
		}
	default:
		println(42)
	}

}
