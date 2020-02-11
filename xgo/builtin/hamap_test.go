package builtin

func test_hamap1() {
	ov := 12345
	hv := htkey_hash_int(&ov, 0)
	println(hv)
	s := "foo"
	hv = htkey_hash_str(s.ptr, s.len)
	println(hv)
}

func test_hamap2() {
	m1 := mirmap_new(Int)
	m1len := m1.len()
	m1cap := m1.cap()
	println(m1len, m1cap)

	k1 := 5
	v1 := 6
	m1.insert(&k1, &v1)
	m1.dump()

	k2 := 6
	v2 := 7
	m1.insert(&k2, &v2)
	m1.dump()

	k3 := 7
	v3 := 8
	m1.insert(&k3, &v3)

	m1.dump()

	k3 = 7
	v3 = 9
	m1.insert(&k3, &v3)

	m1.dump()

	for i := 0; i < 100000; i++ {
		k := i
		v := i
		m1.insert(&k, &v)
	}
	m1.dumpmin()
	m1.chklinked()
	m1.clear()
	m1.dumpmin()
}
