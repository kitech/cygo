package builtin

const uniform_functype_method = 1
const uniform_functype_bare = 2
const uniform_functype_clos = 2

type Unifunc struct {
	kind        int
	obj         voidptr
	under_fnptr func(voidptr)
}

func (ufn *Unifunc) Call() {
	fnptr := ufn.under_fnptr
	switch ufn.kind {
	case uniform_functype_bare:
		fnptr(nil)
	case uniform_functype_method:
		fnptr(ufn.obj)
	case uniform_functype_clos:
		fnptr(ufn.obj)
	}
}
