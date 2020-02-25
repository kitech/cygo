package main

var Err1 error

func foo() (int, string) {
	if true {
		return 1, "true"
	} else {
		return 0, "false"
	}
}

func main() {
	var v = 5
	println(v)

	/*
		catch {
		case Err1:
		}
	*/
}
