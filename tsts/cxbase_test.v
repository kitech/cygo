module tsts

fn test_1() {}

fn test_2() {
	c99 {
		assert(strcmp(IDTSTR(123),"123")==0);
		assert(IDTLEN(abcde) == 6);
		assert(strlen(IDTSTR(abcde)) == 5);

		#define testargcnt(...) PP_NARG(__VA_ARGS__)
		assert(testargcnt(1, 2, "abc") == 3);
		// printf("%d\n", testargcnt()); // 1??
		assert(testargcnt() == 1); // ???
		#undef testargcnt
	}
	c99 {
		assert(ctypeidof(123) == ctypeid_int);
		// printf("%s\n", ctypeid_tostr(ctypeidof(123)));
		assert(strcmp(ctypeid_tostr(ctypeidof(123)), "int")==0);

		// printf("%d, %s\n", ctypeidof(tsts__test_2), ctypeid_tostr(ctypeidof(tsts__test_2)));
		assert(strcmp(ctypeid_tostr(ctypeidof(tsts__test_2)), "void(*)()")==0);
	}
	c99 {		log_info(42, "foo"); }
	// s := C.IDTSTR(123)
}

// structs
fn test_3() {

}

// cxtls
c99 {
	cxtls_def(long, foo);
	cxtls_def(double, foo2);
	void barrr() {
	    long x = cxtls_get(foo);
	    // log_info(x);
		log_errorif(x!=0, "test failed");
	    cxtls_set(foo, 12345);
	    x = cxtls_get(foo);
	    log_info(x);
		log_errorif(x!=12345, x, "test failed");
	    // assert(x == 12345);
	}

	void barrr2() {
	    double x = cxtls_get(foo2);
	    // log_info(x);
		log_errorif(x!=0, "test failed");
	    cxtls_set(foo2, 123.456);
	    x = cxtls_get(foo2);
		printf("foo2=%g\n", x);
	    log_info(x);
		log_errorif(x!=123.456, x, "test failed");
		print_binhex((void*)&x, sizeof(x)+27);

	}
}
fn test_4() {
	c99 { barrr(); barrr2(); }
}
