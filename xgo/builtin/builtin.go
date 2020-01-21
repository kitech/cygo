package builtin

// don't use other packages, only C is supported

/*
#include <stdlib.h>
#include <stdio.h>
*/
import "C"

type Nothing int

func keep() {}

func assert() {}

// func panic()   {}
func panicln() {}
func fatal()   {}
func fatalln() {}

func malloc() voidptr  { return nil }
func realloc() voidptr { return nil }
func free()            {}

func sizeof() int   { return 0 }
func alignof() int  { return 0 }
func offsetof() int { return 0 }
