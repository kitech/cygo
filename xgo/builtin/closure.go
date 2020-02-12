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

type gxcallable struct {
	obj   voidptr
	fnptr voidptr
}

//export gxcallable_new
func gxcallable_new(fnptr voidptr, obj voidptr) voidptr {
	var caobj *gxcallable = malloc3(sizeof(voidptr(0)) * 2)
	caobj.obj = obj
	caobj.fnptr = fnptr
	return caobj
}
