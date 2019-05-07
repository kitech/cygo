package main

type foo func() foo

func g() foo {
	return func() foo {
		return nil
	}
}
