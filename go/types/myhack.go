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
	/*{ // string.join([]string, sep string) int
		var sig *Signature
		recv := NewVar(token.NoPos, nil, "this", Typ[String])
		arg0 := NewVar(token.NoPos, nil, "sep", Typ[String])
		params := NewTuple(arg0)
		r0 := NewVar(token.NoPos, nil, "", NewSlice(Typ[Int]))
		results := NewTuple(r0)
		sig = NewSignature(recv, params, results, false)
		m1 := NewFunc(token.NoPos, nil, "join", sig)
		strmths = append(strmths, m1)
	}*/
	{
		f1 := NewVar(token.NoPos, nil, "len", Typ[Int])
		strflds = append(strflds, f1)
	}
	{
		f1 := NewVar(token.NoPos, nil, "ptr", Typ[Byteptr])
		strflds = append(strflds, f1)
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
	{ // array.len() int
		var sig *Signature
		recv := NewVar(token.NoPos, nil, "this", NewSlice(nil))
		var params *Tuple
		r0 := NewVar(token.NoPos, nil, "", Typ[Int])
		results := NewTuple(r0)
		sig = NewSignature(recv, params, results, false)
		m1 := NewFunc(token.NoPos, nil, "len", sig)
		arrmths = append(arrmths, m1)
	}
	{ // array.cap() int
		var sig *Signature
		recv := NewVar(token.NoPos, nil, "this", NewSlice(nil))
		var params *Tuple
		r0 := NewVar(token.NoPos, nil, "", Typ[Int])
		results := NewTuple(r0)
		sig = NewSignature(recv, params, results, false)
		m1 := NewFunc(token.NoPos, nil, "cap", sig)
		arrmths = append(arrmths, m1)
	}
	{ // array.ptr() voidptr
		var sig *Signature
		recv := NewVar(token.NoPos, nil, "this", NewSlice(nil))
		var params *Tuple
		r0 := NewVar(token.NoPos, nil, "", Typ[Voidptr])
		results := NewTuple(r0)
		sig = NewSignature(recv, params, results, false)
		m1 := NewFunc(token.NoPos, nil, "ptr", sig)
		arrmths = append(arrmths, m1)
	}
	{ // array.append()
		var sig *Signature
		recv := NewVar(token.NoPos, nil, "this", NewSlice(nil))
		arg0 := NewVar(token.NoPos, nil, "elem", Typ[Voidptr])
		params := NewTuple(arg0)
		r0 := NewVar(token.NoPos, nil, "", NewSlice(nil))
		results := NewTuple(r0)
		sig = NewSignature(recv, params, results, false)
		m1 := NewFunc(token.NoPos, nil, "append", sig)
		arrmths = append(arrmths, m1)
	}
	{ // array.reverse()
		var sig *Signature
		recv := NewVar(token.NoPos, nil, "this", NewSlice(nil))
		// arg0 := NewVar(token.NoPos, nil, "elem", Typ[Voidptr])
		// params := NewTuple(arg0)
		var params *Tuple
		r0 := NewVar(token.NoPos, nil, "", NewSlice(nil))
		results := NewTuple(r0)
		sig = NewSignature(recv, params, results, false)
		m1 := NewFunc(token.NoPos, nil, "reverse", sig)
		arrmths = append(arrmths, m1)
	}
	{ // array.delete()
		var sig *Signature
		recv := NewVar(token.NoPos, nil, "this", NewSlice(nil))
		arg0 := NewVar(token.NoPos, nil, "index", Typ[Int])
		params := NewTuple(arg0)
		r0 := NewVar(token.NoPos, nil, "", NewSlice(nil))
		results := NewTuple(r0)
		sig = NewSignature(recv, params, results, false)
		m1 := NewFunc(token.NoPos, nil, "delete", sig)
		arrmths = append(arrmths, m1)
	}
	{ // array.clear()
		var sig *Signature
		recv := NewVar(token.NoPos, nil, "this", NewSlice(nil))
		// arg0 := NewVar(token.NoPos, nil, "elem", Typ[Voidptr])
		// params := NewTuple(arg0)
		var params *Tuple
		r0 := NewVar(token.NoPos, nil, "", NewSlice(nil))
		results := NewTuple(r0)
		sig = NewSignature(recv, params, results, false)
		m1 := NewFunc(token.NoPos, nil, "clear", sig)
		arrmths = append(arrmths, m1)
	}
	{ // array.join() string
		var sig *Signature
		recv := NewVar(token.NoPos, nil, "this", NewSlice(Typ[String]))
		var params *Tuple
		r0 := NewVar(token.NoPos, nil, "", Typ[String])
		results := NewTuple(r0)
		sig = NewSignature(recv, params, results, false)
		m1 := NewFunc(token.NoPos, nil, "join", sig)
		arrmths = append(arrmths, m1)
	}
	{ // array.map() []string
		var sig *Signature
		recv := NewVar(token.NoPos, nil, "this", NewSlice(Typ[String]))
		var params *Tuple
		r0 := NewVar(token.NoPos, nil, "", Typ[String])
		results := NewTuple(r0)
		sig = NewSignature(recv, params, results, false)
		m1 := NewFunc(token.NoPos, nil, "map", sig)
		arrmths = append(arrmths, m1)
	}
	{ // array.filter() []string
		var sig *Signature
		recv := NewVar(token.NoPos, nil, "this", NewSlice(Typ[String]))
		var params *Tuple
		r0 := NewVar(token.NoPos, nil, "", Typ[String])
		results := NewTuple(r0)
		sig = NewSignature(recv, params, results, false)
		m1 := NewFunc(token.NoPos, nil, "filter", sig)
		arrmths = append(arrmths, m1)
	}

	{ // map.len() int
		var sig *Signature
		recv := NewVar(token.NoPos, nil, "this", NewMap(nil, nil))
		var params *Tuple
		r0 := NewVar(token.NoPos, nil, "", Typ[Int])
		results := NewTuple(r0)
		sig = NewSignature(recv, params, results, false)
		m1 := NewFunc(token.NoPos, nil, "len", sig)
		mapmths = append(mapmths, m1)
	}
	{ // map.cap() int
		var sig *Signature
		recv := NewVar(token.NoPos, nil, "this", NewMap(nil, nil))
		var params *Tuple
		r0 := NewVar(token.NoPos, nil, "", Typ[Int])
		results := NewTuple(r0)
		sig = NewSignature(recv, params, results, false)
		m1 := NewFunc(token.NoPos, nil, "cap", sig)
		mapmths = append(mapmths, m1)
	}
	{ // map.haskey() bool
		var sig *Signature
		recv := NewVar(token.NoPos, nil, "this", NewMap(nil, nil))
		var params *Tuple
		r0 := NewVar(token.NoPos, nil, "", Typ[Bool])
		results := NewTuple(r0)
		sig = NewSignature(recv, params, results, false)
		m1 := NewFunc(token.NoPos, nil, "haskey", sig)
		mapmths = append(mapmths, m1)
	}

	{ // byte.isspace() bool
		var sig *Signature
		recv := NewVar(token.NoPos, nil, "this", Typ[Byte])
		var params *Tuple
		r0 := NewVar(token.NoPos, nil, "", Typ[Bool])
		results := NewTuple(r0)
		sig = NewSignature(recv, params, results, false)
		m1 := NewFunc(token.NoPos, nil, "isspace", sig)
		intmths = append(intmths, m1)
	}
	{ // byte.isdigit() bool
		var sig *Signature
		recv := NewVar(token.NoPos, nil, "this", Typ[Byte])
		var params *Tuple
		r0 := NewVar(token.NoPos, nil, "", Typ[Bool])
		results := NewTuple(r0)
		sig = NewSignature(recv, params, results, false)
		m1 := NewFunc(token.NoPos, nil, "isdigit", sig)
		intmths = append(intmths, m1)
	}
	{ // byte.isnumber() bool
		var sig *Signature
		recv := NewVar(token.NoPos, nil, "this", Typ[Byte])
		var params *Tuple
		r0 := NewVar(token.NoPos, nil, "", Typ[Bool])
		results := NewTuple(r0)
		sig = NewSignature(recv, params, results, false)
		m1 := NewFunc(token.NoPos, nil, "isnumber", sig)
		intmths = append(intmths, m1)
	}

	{
		for _, ty := range []BasicKind{
			Bool, Int, Int8, Int16, Int32, Int64,
			Uint, Uint8, Uint16, Uint32, Uint64,
			Uintptr, Float32, Float64,
			Byte, Rune, Usize, Voidptr, Byteptr, Charptr} {
			var sig *Signature
			recv := NewVar(token.NoPos, nil, "this", Typ[ty])
			var params *Tuple
			r0 := NewVar(token.NoPos, nil, "", Typ[String])
			results := NewTuple(r0)
			sig = NewSignature(recv, params, results, false)
			m1 := NewFunc(token.NoPos, nil, "repr", sig)
			intmths = append(intmths, m1)
		}
	}

	println(len(strmths), len(arrmths), len(mapmths))
}

var strmths = []*Func{}
var strflds = []*Var{}
var arrmths = []*Func{}
var arrflds = []*Var{}
var mapmths = []*Func{}
var intmths = []*Func{}
var builtin_type_methods = map[Type][]*Func{}

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
