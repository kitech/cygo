package types

import "log"

// too late
func HackExtraBuiltin() {
	if true {
		return
	}
	tybno := UntypedNil
	tybinfo := IsUntyped
	{
		vptrty := &Basic{tybno << 1, tybinfo << 1, "voidptr"}
		Typ = append(Typ, vptrty)
	}
	{
		vptrty := &Basic{tybno << 2, tybinfo << 2, "byteptr"}
		Typ = append(Typ, vptrty)
	}
	log.Println("222222222")
}

// Nil represents the predeclared value nil.
type Nilptr struct {
	object
}

const ctypebase = 200

var ctypeno int = ctypebase
var ctypetys = map[string]Type{}

func NewCtype(tyname string) Type {
	if ty, ok := ctypetys[tyname]; ok {
		return ty
	}
	no := ctypeno
	ctypeno++

	ty := &Basic{}
	ty.name = tyname
	ty.kind = BasicKind(no)
	ty.info = BasicInfo(no) | IsOrdered | IsNumeric
	ty.info = BasicInfo(no) | IsOrdered
	ty.info = BasicInfo(no) | IsNumeric
	ctypetys[tyname] = ty

	return ty
}

func isCdefType(typ Type) bool {
	t, ok := typ.Underlying().(*Basic)
	return ok && t.kind >= ctypebase
}
func isVoidptr(typ Type) bool {
	// TODO(gri): Is this (typ.Underlying() instead of just typ) correct?
	//            The spec does not say so, but gc claims it is. See also
	//            issue 6326.
	t, ok := typ.Underlying().(*Basic)
	return ok && t.kind == Voidptr
}
func isByteptr(typ Type) bool {
	// TODO(gri): Is this (typ.Underlying() instead of just typ) correct?
	//            The spec does not say so, but gc claims it is. See also
	//            issue 6326.
	t, ok := typ.Underlying().(*Basic)
	return ok && t.kind == Byteptr
}

func TypeAlias() []*Basic { return aliases[:] }
