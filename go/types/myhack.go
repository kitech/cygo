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
	{ // string.cstr() byteptr
		var sig *Signature
		recv := NewVar(token.NoPos, nil, "this", Typ[String])
		var params *Tuple
		r0 := NewVar(token.NoPos, nil, "", Typ[Byteptr])
		results := NewTuple(r0)
		sig = NewSignature(recv, params, results, false)
		m1 := NewFunc(token.NoPos, nil, "cstr", sig)
		strmths = append(strmths, m1)
	}
	{ // string.split(sep string) []string
		var sig *Signature
		recv := NewVar(token.NoPos, nil, "this", Typ[String])
		arg0 := NewVar(token.NoPos, nil, "sep", Typ[String])
		params := NewTuple(arg0)
		r0 := NewVar(token.NoPos, nil, "", NewSlice(Typ[String]))
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
	{ // string.index(sep string) int
		var sig *Signature
		recv := NewVar(token.NoPos, nil, "this", Typ[String])
		arg0 := NewVar(token.NoPos, nil, "sep", Typ[String])
		params := NewTuple(arg0)
		r0 := NewVar(token.NoPos, nil, "", Typ[Int])
		results := NewTuple(r0)
		sig = NewSignature(recv, params, results, false)
		m1 := NewFunc(token.NoPos, nil, "index", sig)
		strmths = append(strmths, m1)
	}
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
	{ // string.left(sep string) string
		var sig *Signature
		recv := NewVar(token.NoPos, nil, "this", Typ[String])
		arg0 := NewVar(token.NoPos, nil, "sep", Typ[String])
		params := NewTuple(arg0)
		r0 := NewVar(token.NoPos, nil, "", Typ[String])
		results := NewTuple(r0)
		sig = NewSignature(recv, params, results, false)
		m1 := NewFunc(token.NoPos, nil, "left", sig)
		strmths = append(strmths, m1)
	}
	{ // string.right(sep string) string
		var sig *Signature
		recv := NewVar(token.NoPos, nil, "this", Typ[String])
		arg0 := NewVar(token.NoPos, nil, "sep", Typ[String])
		params := NewTuple(arg0)
		r0 := NewVar(token.NoPos, nil, "", Typ[String])
		results := NewTuple(r0)
		sig = NewSignature(recv, params, results, false)
		m1 := NewFunc(token.NoPos, nil, "right", sig)
		strmths = append(strmths, m1)
	}
	{ // string.prefixed(sep string) bool
		var sig *Signature
		recv := NewVar(token.NoPos, nil, "this", Typ[String])
		arg0 := NewVar(token.NoPos, nil, "sep", Typ[String])
		params := NewTuple(arg0)
		r0 := NewVar(token.NoPos, nil, "", Typ[Bool])
		results := NewTuple(r0)
		sig = NewSignature(recv, params, results, false)
		m1 := NewFunc(token.NoPos, nil, "prefixed", sig)
		strmths = append(strmths, m1)
	}
	{ // string.suffixed(sep string) bool
		var sig *Signature
		recv := NewVar(token.NoPos, nil, "this", Typ[String])
		arg0 := NewVar(token.NoPos, nil, "sep", Typ[String])
		params := NewTuple(arg0)
		r0 := NewVar(token.NoPos, nil, "", Typ[Bool])
		results := NewTuple(r0)
		sig = NewSignature(recv, params, results, false)
		m1 := NewFunc(token.NoPos, nil, "suffixed", sig)
		strmths = append(strmths, m1)
	}
	{ // string.empty() bool
		var sig *Signature
		recv := NewVar(token.NoPos, nil, "this", Typ[String])
		var params *Tuple
		r0 := NewVar(token.NoPos, nil, "", Typ[Bool])
		results := NewTuple(r0)
		sig = NewSignature(recv, params, results, false)
		m1 := NewFunc(token.NoPos, nil, "empty", sig)
		strmths = append(strmths, m1)
	}
	{ // string.repeat(count int) string
		var sig *Signature
		recv := NewVar(token.NoPos, nil, "this", Typ[String])
		arg0 := NewVar(token.NoPos, nil, "count", Typ[Int])
		params := NewTuple(arg0)
		r0 := NewVar(token.NoPos, nil, "", Typ[String])
		results := NewTuple(r0)
		sig = NewSignature(recv, params, results, false)
		m1 := NewFunc(token.NoPos, nil, "repeat", sig)
		strmths = append(strmths, m1)
	}
	{ // string.replace(old string, new string, count int) string
		var sig *Signature
		recv := NewVar(token.NoPos, nil, "this", Typ[String])
		arg0 := NewVar(token.NoPos, nil, "old", Typ[String])
		arg1 := NewVar(token.NoPos, nil, "new", Typ[String])
		arg2 := NewVar(token.NoPos, nil, "count", Typ[Int])
		params := NewTuple(arg0, arg1, arg2)
		r0 := NewVar(token.NoPos, nil, "", Typ[String])
		results := NewTuple(r0)
		sig = NewSignature(recv, params, results, false)
		m1 := NewFunc(token.NoPos, nil, "replace", sig)
		strmths = append(strmths, m1)
	}
	{ // string.replaceall(old string, new string) string
		var sig *Signature
		recv := NewVar(token.NoPos, nil, "this", Typ[String])
		arg0 := NewVar(token.NoPos, nil, "old", Typ[String])
		arg1 := NewVar(token.NoPos, nil, "new", Typ[String])
		params := NewTuple(arg0, arg1)
		r0 := NewVar(token.NoPos, nil, "", Typ[String])
		results := NewTuple(r0)
		sig = NewSignature(recv, params, results, false)
		m1 := NewFunc(token.NoPos, nil, "replaceall", sig)
		strmths = append(strmths, m1)
	}
	{ // string.toupper() string
		var sig *Signature
		recv := NewVar(token.NoPos, nil, "this", Typ[String])
		arg0 := NewVar(token.NoPos, nil, "sep", Typ[String])
		params := NewTuple(arg0)
		r0 := NewVar(token.NoPos, nil, "", Typ[String])
		results := NewTuple(r0)
		sig = NewSignature(recv, params, results, false)
		m1 := NewFunc(token.NoPos, nil, "toupper", sig)
		strmths = append(strmths, m1)
	}
	{ // string.tolower() string
		var sig *Signature
		recv := NewVar(token.NoPos, nil, "this", Typ[String])
		var params *Tuple
		r0 := NewVar(token.NoPos, nil, "", Typ[String])
		results := NewTuple(r0)
		sig = NewSignature(recv, params, results, false)
		m1 := NewFunc(token.NoPos, nil, "tolower", sig)
		strmths = append(strmths, m1)
	}
	{ // string.totitle() string
		var sig *Signature
		recv := NewVar(token.NoPos, nil, "this", Typ[String])
		var params *Tuple
		r0 := NewVar(token.NoPos, nil, "", Typ[String])
		results := NewTuple(r0)
		sig = NewSignature(recv, params, results, false)
		m1 := NewFunc(token.NoPos, nil, "totitle", sig)
		strmths = append(strmths, m1)
	}
	{ // string.tohex() string
		var sig *Signature
		recv := NewVar(token.NoPos, nil, "this", Typ[String])
		var params *Tuple
		r0 := NewVar(token.NoPos, nil, "", Typ[String])
		results := NewTuple(r0)
		sig = NewSignature(recv, params, results, false)
		m1 := NewFunc(token.NoPos, nil, "tohex", sig)
		strmths = append(strmths, m1)
	}
	{ // string.tomd5() string
		var sig *Signature
		recv := NewVar(token.NoPos, nil, "this", Typ[String])
		var params *Tuple
		r0 := NewVar(token.NoPos, nil, "", Typ[String])
		results := NewTuple(r0)
		sig = NewSignature(recv, params, results, false)
		m1 := NewFunc(token.NoPos, nil, "tomd5", sig)
		strmths = append(strmths, m1)
	}
	{ // string.tosha1() string
		var sig *Signature
		recv := NewVar(token.NoPos, nil, "this", Typ[String])
		var params *Tuple
		r0 := NewVar(token.NoPos, nil, "", Typ[String])
		results := NewTuple(r0)
		sig = NewSignature(recv, params, results, false)
		m1 := NewFunc(token.NoPos, nil, "tosha1", sig)
		strmths = append(strmths, m1)
	}
	{ // string.tof32() f32
		var sig *Signature
		recv := NewVar(token.NoPos, nil, "this", Typ[String])
		var params *Tuple
		r0 := NewVar(token.NoPos, nil, "", Typ[Float32])
		results := NewTuple(r0)
		sig = NewSignature(recv, params, results, false)
		m1 := NewFunc(token.NoPos, nil, "tof32", sig)
		strmths = append(strmths, m1)
	}
	{ // string.tof64() f64
		var sig *Signature
		recv := NewVar(token.NoPos, nil, "this", Typ[String])
		var params *Tuple
		r0 := NewVar(token.NoPos, nil, "", Typ[Float64])
		results := NewTuple(r0)
		sig = NewSignature(recv, params, results, false)
		m1 := NewFunc(token.NoPos, nil, "tof64", sig)
		strmths = append(strmths, m1)
	}
	{ // string.toint() int
		var sig *Signature
		recv := NewVar(token.NoPos, nil, "this", Typ[String])
		var params *Tuple
		r0 := NewVar(token.NoPos, nil, "", Typ[Int])
		results := NewTuple(r0)
		sig = NewSignature(recv, params, results, false)
		m1 := NewFunc(token.NoPos, nil, "toint", sig)
		strmths = append(strmths, m1)
	}
	{ // string.isdigit() bool
		var sig *Signature
		recv := NewVar(token.NoPos, nil, "this", Typ[String])
		var params *Tuple
		r0 := NewVar(token.NoPos, nil, "", Typ[Bool])
		results := NewTuple(r0)
		sig = NewSignature(recv, params, results, false)
		m1 := NewFunc(token.NoPos, nil, "isdigit", sig)
		strmths = append(strmths, m1)
	}
	{ // string.isnumber() bool
		var sig *Signature
		recv := NewVar(token.NoPos, nil, "this", Typ[String])
		var params *Tuple
		r0 := NewVar(token.NoPos, nil, "", Typ[Bool])
		results := NewTuple(r0)
		sig = NewSignature(recv, params, results, false)
		m1 := NewFunc(token.NoPos, nil, "isnumber", sig)
		strmths = append(strmths, m1)
	}
	{ // string.isprintable() bool
		var sig *Signature
		recv := NewVar(token.NoPos, nil, "this", Typ[String])
		var params *Tuple
		r0 := NewVar(token.NoPos, nil, "", Typ[Bool])
		results := NewTuple(r0)
		sig = NewSignature(recv, params, results, false)
		m1 := NewFunc(token.NoPos, nil, "isprintable", sig)
		strmths = append(strmths, m1)
	}
	{ // string.ishex() bool
		var sig *Signature
		recv := NewVar(token.NoPos, nil, "this", Typ[String])
		var params *Tuple
		r0 := NewVar(token.NoPos, nil, "", Typ[Bool])
		results := NewTuple(r0)
		sig = NewSignature(recv, params, results, false)
		m1 := NewFunc(token.NoPos, nil, "ishex", sig)
		strmths = append(strmths, m1)
	}
	{ // string.isascii() bool
		var sig *Signature
		recv := NewVar(token.NoPos, nil, "this", Typ[String])
		var params *Tuple
		r0 := NewVar(token.NoPos, nil, "", Typ[Bool])
		results := NewTuple(r0)
		sig = NewSignature(recv, params, results, false)
		m1 := NewFunc(token.NoPos, nil, "isascii", sig)
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

	{
		for _, ty := range []BasicKind{
			Bool, Int, Int8, Int16, Int32, Int64,
			Uint, Uint8, Uint16, Uint32, Uint64,
			Uintptr,
			Float32, Float64,
			Rune, Usize,
			Voidptr, Byteptr, Charptr} {
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
