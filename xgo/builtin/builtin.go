package builtin

// don't use other packages, only C is supported

/*
#include <stdlib.h>
#include <stdio.h>
*/
import "C"

func keep() {}

func assert() {}

// func panic()   {}
func panicln() {}
func fatal()   {}
func fatalln() {}

func malloc()  {}
func realloc() {}
func free()    {}

func sizeof()   {}
func alignof()  {}
func offsetof() {}
