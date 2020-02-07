package main

func foo() (int, error) {
	return 0, nil
}

func foo2() (int, error) {
	var err error
	errreturn(err, 1, err)
	return
}

func foo3() (string, error) {
	var err error
	errreturn(err, "abc", err)
	return
}

func foo4() error {
	var err error
	errreturn(err, err)
	return
}

func main() {
	foo()

	v1, err1 := foo()
	errreturn(err1)

	v2, err2 := foo()

	switch {
	case 1 == 2:
	case 3 == 4:
	}
}
