package main

type myerror struct {
}

func (err *myerror) Error() string {
	return "myerrored"
}

func main() {
	var err error
	err = &myerror{}

	err2 := err.(*myerror)
	println(err2.Error())

	// err2, ok := err.(*myerror)
}
