package types

import (
	"go/token"
)

const (
	Voidptr = UntypedNil + 1
	Byteptr = UntypedNil + 2
	Charptr = UntypedNil + 3
	Wideptr = UntypedNil + 4

	// aliases

	Usize = Uintptr
	F32   = Float32
	F64   = Float64
	U64   = Uint64
	I64   = Int64
	U32   = Uint32
	I32   = Int32
	U16   = Uint16
	I16   = Int16
	U8    = Uint8
	I8    = Int8
)

func init() {
	// too early
	// HackExtraBuiltin()
}

// too late, call in universe.go:190 init(), before defPredxxx
func HackExtraBuiltin() {
	hetyp := []*Basic{
		{Voidptr, IsPointer | IsBoolean | IsNumeric, "voidptr"},
		{Byteptr, IsPointer | IsBoolean | IsNumeric, "byteptr"},
		{Charptr, IsPointer | IsBoolean | IsNumeric, "charptr"},
		{Wideptr, IsPointer | IsBoolean, "wideptr"},
	}
	for _, typ := range hetyp {
		Typ = append(Typ, typ)
	}

	// modify var aliases = [...] => []
	healias := []*Basic{
		{Usize, IsInteger | IsUnsigned | IsPointer, "usize"},
		{F32, IsFloat, "f32"},
		{F64, IsFloat, "f64"},
		{U64, IsInteger | IsUnsigned, "u64"},
		{I64, IsInteger, "i64"},
		{U32, IsInteger | IsUnsigned, "u32"},
		{I32, IsInteger, "i32"},
		{U16, IsInteger | IsUnsigned, "u16"},
		{I16, IsInteger, "i16"},
	}
	for _, typ := range healias {
		aliases = append(aliases, typ)
	}

	def(&Nilofc{object{name: "nilofc", typ: Typ[UntypedNil], color_: black}})
	def(&Cnil{object{name: "cnil", typ: Typ[UntypedNil], color_: black}})
	def(&Cnull{object{name: "cnull", typ: Typ[UntypedNil], color_: black}})
	// log.Println("222222222")

	fillBasicMethods()
}

func fillBasicMethods() {
	{
		f1 := NewVar(token.NoPos, nil, "len", Typ[Int])
		strflds = append(strflds, f1)
		builtin_type_fields[Typ[String]] = append(builtin_type_fields[Typ[String]], f1)
	}
	{
		f1 := NewVar(token.NoPos, nil, "ptr", Typ[Byteptr])
		strflds = append(strflds, f1)
		builtin_type_fields[Typ[String]] = append(builtin_type_fields[Typ[String]], f1)
	}
	{
		f1 := NewVar(token.NoPos, nil, "len", Typ[Int])
		arrflds = append(arrflds, f1)
	}
	{
		f1 := NewVar(token.NoPos, nil, "cap", Typ[Int])
		arrflds = append(arrflds, f1)
	}
	{
		f1 := NewVar(token.NoPos, nil, "ptr", Typ[Voidptr])
		arrflds = append(arrflds, f1)
	}
	{
		// f1 := NewVar(token.NoPos, nil, "len", Typ[Int])
		// mapflds = append(arrflds, f1)
	}
	{
		// f1 := NewVar(token.NoPos, nil, "cap", Typ[Int])
		// mapflds = append(arrflds, f1)
	}

	println(len(strmths), len(arrmths), len(mapmths))
}

var strmths = []*Func{}
var strflds = []*Var{}
var arrmths = []*Func{}
var arrflds = []*Var{}
var mapmths = []*Func{}
var mapflds = []*Var{}
var intmths = []*Func{}
var builtin_type_methods = map[Type][]*Func{}
var builtin_type_fields = map[Type][]*Var{}

func DumpBuiltinMethods() map[string]int {
	res := map[string]int{
		"strmths": len(strmths), "arrmths": len(arrmths),
		"mapmths": len(mapmths), "intmths": len(intmths),
		"sums": len(builtin_type_methods)}
	return res
}
func AddBuiltinMethod(typ Type, fnty *Func) {
	switch typ2 := typ.(type) {
	case *Basic:
		switch typ2.kind {
		case String:
			strmths = append(strmths, fnty)
		}
		builtin_type_methods[typ] = append(builtin_type_methods[typ], fnty)
	case *Map:
		mapmths = append(mapmths, fnty)
		builtin_type_methods[typ] = append(builtin_type_methods[typ], fnty)
	case *Slice:
		arrmths = append(arrmths, fnty)
		builtin_type_methods[typ] = append(builtin_type_methods[typ], fnty)
	default:
	}
}

// Nil represents the predeclared value nil.
type Nilofc struct {
	object
}
type Cnil struct {
	object
}
type Cnull struct {
	object
}

func TypeAlias() []*Basic { return aliases[:] }

///
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
	ty.info = BasicInfo(no) | IsOrdered | IsNumeric | IsPointer
	// ty.info = BasicInfo(no) | IsOrdered
	// ty.info = BasicInfo(no) | IsNumeric
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
func isCharptr(typ Type) bool {
	// TODO(gri): Is this (typ.Underlying() instead of just typ) correct?
	//            The spec does not say so, but gc claims it is. See also
	//            issue 6326.
	t, ok := typ.Underlying().(*Basic)
	return ok && t.kind == Charptr
}
