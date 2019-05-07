package foo

import (
	"fmt"
	"time"
)

func bar() (int, error) {
	fmt.Println(time.Now())
	foo()
	return 0, nil
}

func foo() []string {
	return nil
}
