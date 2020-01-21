package builtin

// don't use other packages, only C is supported

/*
#include <stdlib.h>
#include <stdio.h>
*/
import "C"

func keep() {}

// func panic()   {}
func panicln() {}
func fatal()   {}
func fatalln() {}

func malloc() voidptr  { return nil }
func realloc() voidptr { return nil }
func free()            {}

//[nomangle]
func assert()
func sizeof() int
func alignof() int
func offsetof() int
