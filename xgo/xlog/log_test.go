package xlog

func test_1() {
	test_addr2line()
	printint(123)
	// xlog.printstr("str123")
	// xlog.printptr((voidptr)(0x123))
	// xlog.printflt(123.456)
	printx(123)
	printx(123.456)
	var p1 voidptr = 0x5
	printx(p1)
	printx("eee")
	printx('k')
	rv := xlog.printx1(true, "eee", 123, 123.456, 'k')
}
