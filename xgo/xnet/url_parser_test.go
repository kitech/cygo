package xnet

func test_urlpr1() {
	uo := ParseUrl("https://www.google.com/foo")
	println(uo)
	// xlog.Println(uo.Scheme, uo.Host, uo.Path)
}
