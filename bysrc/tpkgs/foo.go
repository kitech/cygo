package foo

import (
	"fmt"
	"time"
)

func bar() int {
	fmt.Println(time.Now())
	foo()
	go foo()
	return 0
}

func foo() []string {
	return nil
}

func foo1() string {
	return ""
}
