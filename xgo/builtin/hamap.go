package builtin

type mirmap struct {
	len_ int
	cap_ int

	key voidptr
	val voidptr

	keysz int
	valsz int
}

func mirmap_new() *mirmap {
	mp := &mirmap{}
	return mp
}

func (mp *mirmap) dummy() {

}

//export mirmap_dummy2
func (mp *mirmap) dummy2() {}

func (mp *mirmap) each(fn func(key voidptr, val voidptr)) {

}

func (mp *mirmap) mapfn(fn func(key voidptr, val voidptr) *mirmap) {

}

func (mp *mirmap) reduce(fn func(key voidptr, val voidptr) bool) {

}

func (mp *mirmap) haskey(k voidptr) bool {
	return false
}

func (mp *mirmap) len() int {
	return 0
}

func (mp *mirmap) cap() int {
	return 0
}
