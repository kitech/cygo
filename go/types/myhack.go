package types

import "go/token"

const (
	Voidptr = UntypedNil + 1
	Byteptr = UntypedNil + 2
	Charptr = UntypedNil + 3

	// aliases

	Usize = Uintptr
	F32   = Float32
	F64   = Float64
	U64   = Uint64
	I64   = Int64
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
		{Voidptr, IsPointer | IsBoolean, "voidptr"},
		{Byteptr, IsPointer | IsBoolean, "byteptr"},
		{Charptr, IsPointer | IsBoolean, "charptr"},
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
	{ // string.len() int
		var sig *Signature
		recv := NewVar(token.NoPos, nil, "this", Typ[String])
		var params *Tuple
		r0 := NewVar(token.NoPos, nil, "", Typ[Int])
		results := NewTuple(r0)
		sig = NewSignature(recv, params, results, false)
		m1 := NewFunc(token.NoPos, nil, "len", sig)
		strmths = append(strmths, m1)
	}
	{ // string.split(sep string) []string
		var sig *Signature
		recv := NewVar(token.NoPos, nil, "this", Typ[String])
		arg0 := NewVar(token.NoPos, nil, "sep", Typ[String])
		params := NewTuple(arg0)
		r0 := NewVar(token.NoPos, nil, "", NewSlice(Typ[Int]))
		results := NewTuple(r0)
		sig = NewSignature(recv, params, results, false)
		m1 := NewFunc(token.NoPos, nil, "split", sig)
		strmths = append(strmths, m1)
	}
	{ // string.trimsp() string
		var sig *Signature
		recv := NewVar(token.NoPos, nil, "this", Typ[String])
		var params *Tuple
		r0 := NewVar(token.NoPos, nil, "", Typ[String])
		results := NewTuple(r0)
		sig = NewSignature(recv, params, results, false)
		m1 := NewFunc(token.NoPos, nil, "trimsp", sig)
		strmths = append(strmths, m1)
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

	{ // int.repr() string
		var sig *Signature
		recv := NewVar(token.NoPos, nil, "this", Typ[Int])
		var params *Tuple
		r0 := NewVar(token.NoPos, nil, "", Typ[String])
		results := NewTuple(r0)
		sig = NewSignature(recv, params, results, false)
		m1 := NewFunc(token.NoPos, nil, "repr", sig)
		intmths = append(intmths, m1)
	}
	{ // float32.repr() string
		var sig *Signature
		recv := NewVar(token.NoPos, nil, "this", Typ[Float32])
		var params *Tuple
		r0 := NewVar(token.NoPos, nil, "", Typ[String])
		results := NewTuple(r0)
		sig = NewSignature(recv, params, results, false)
		m1 := NewFunc(token.NoPos, nil, "repr", sig)
		intmths = append(intmths, m1)
	}
	{ // float64.repr() string
		var sig *Signature
		recv := NewVar(token.NoPos, nil, "this", Typ[Float64])
		var params *Tuple
		r0 := NewVar(token.NoPos, nil, "", Typ[String])
		results := NewTuple(r0)
		sig = NewSignature(recv, params, results, false)
		m1 := NewFunc(token.NoPos, nil, "repr", sig)
		intmths = append(intmths, m1)
	}

	println(len(strmths), len(arrmths), len(mapmths))
}

var strmths = []*Func{}
var arrmths = []*Func{}
var mapmths = []*Func{}
var intmths = []*Func{}

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
