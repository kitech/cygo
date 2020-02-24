package builtin

/*
   extern void* cxmalloc(size_t);
*/
import "C"
import "unsafe"

type BasicKind int

const (
	Invalid = /*BasicKind*/ iota // type is invalid

	Bool
	Int
	Int8
	Int16
	Int32
	Int64
	Uint
	Uint8
	Uint16
	Uint32
	Uint64
	Uintptr
	Float32
	Float64
	Complex64
	Complex128
	// String
	// UnsafePointer

	// types for untyped values
	// UntypedBool
	// UntypedInt
	// UntypedRune
	// UntypedFloat
	// UntypedComplex
	// UntypedString
	// UntypedNil
)

const (
	Array = 17
	Chan
	Func
	Interface
	Map
	Ptr
	Slice
	String
	Struct
	UnsafePointer
)

const (
	// yaUnsafePointer = iota + UnsafePointer // TODO compiler
	yaUnsafePointer = iota + 26
	Voidptr
	Byteptr
	Charptr
	Wideptr
)

const (
	truestr  = "true"
	falsestr = "false"
)

const (
	ctrue  = 1
	cfalse = 0
)

type Metatype struct {
	Size       int      // type size
	Ptrdata    uintptr  // size of memory prefix holding all pointers
	Hash       uint32   // hash of type; avoids computation in hash tables
	Tflag      uint8    // tflag   // extra type information flags
	Align      uint8    // alignment of variable with this type
	FieldAlign uint8    // alignment of struct field with this type
	Kind       uint8    // enumeration for C
	Alg        *typealg // *typeAlg // algorithm table
	Gcdata     byteptr  // garbage collection data
	Str        byteptr  // nameOff // string form
	thisptr    voidptr  // typeOff // type for pointer to this type, may be zero

	elemty *Metatype // for map/array/slice/ptr
	keyty  *Metatype // for map
	count1 uint8     // for uncommon type, like map/slice/ptr
	count2 uint8
	// if use this syntax, need use addr of it, like &extptr
	// extptr *voidptr // generate to: voidptr* exptr, not work
	extptr [0]voidptr // generate to: voidptr [], works
}

type maptype struct {
	typ   Metatype
	flags uint32
}

type arraytype struct {
	typ  Metatype
	elem *Metatype
	len  uintptr
}

type chantype struct {
	typ  Metatype
	elem *Metatype
	dir  uintptr
}

type slicetype struct {
	typ  Metatype
	elem *Metatype
}

type functype struct {
	typ    Metatype
	incnt  uint16
	outcnt uint16
}

type ptrtype struct {
	typ  Metatype
	elem *Metatype
}

type structfield struct {
	name       byteptr
	typ        *Metatype
	offsetAnon uintptr
}

type structtype struct {
	typ     Metatype
	pkgpath byteptr
	fields  []structfield
}

type interfacetype struct {
	typ     Metatype
	pkgpath byteptr
	mhdr    voidptr // []imethod
}

type Eface struct {
	Type *Metatype
	Data *voidptr
}

func Eface_new(typ *Metatype, data voidptr) *Eface {
	efc := &Eface{}
	efc.Type = typ
	efc.Data = data
	return efc
}

func (efc *Eface) Kind() int {
	return efc.Type.Kind
}
func (efc *Eface) Size() int {
	return efc.Type.Size
}
func (efc *Eface) Name() string {
	return gostring(efc.Type.Str)
}
func (efc *Eface) New() voidptr {
	return malloc3(efc.Type.Size)
}

func (efc *Eface) tointp() *int {
	// add unsafe.Pointer() convert, then got valid type
	p := (*int)(unsafe.Pointer(*efc.Data))
	return p
}
func (efc *Eface) Toint() int {
	p := efc.tointp()
	p = (*int)(unsafe.Pointer(*efc.Data))
	return *p
}

// data is pointer of orignal var

//export cxrt_type2eface
func type2eface(mtype voidptr, data voidptr) *Eface {
	var mty *Metatype = mtype

	tyname := gostring(mty.Str)
	if mty.Kind == Struct && tyname == "builtin__mirmap" {
		return type2eface_map(mty, data)
	} else if mty.Kind == Struct && tyname == "builtin__cxarray3" {
		return type2eface_array(mty, data)
	}

	efc := &Eface{}
	efc.Type = mtype
	efc.Data = data
	efc.Data = memdup3(data, mty.Size)
	return efc
}

func type2eface_map(mtype *Metatype, data voidptr) *Eface {
	var mapobjpp **mirmap = data
	var mapobj *mirmap = *mapobjpp
	var newmty *Metatype
	newmty = memdup3(mtype, sizeof(*mtype))
	newmty.Kind = Map
	newmty.thisptr = data

	var keytyp *Metatype
	var valtyp *Metatype

	keytyp = metatype_bykind(mapobj.keykind)
	if keytyp == nil {
		switch mapobj.keykind {
		case Struct:
		}
	}
	assert(keytyp != nil)

	valtyp = metatype_bykind(mapobj.valkind)
	if valtyp == nil {
		switch mapobj.valkind {
		case Struct:

		}
	}
	assert(valtyp != nil)

	newmty.elemty = valtyp
	newmty.keyty = keytyp

	efc := Eface_new(newmty, data)
	return efc
}
func type2eface_array(mtype *Metatype, data voidptr) *Eface {
	var arrobjpp **cxarray3 = data
	var arrobj *cxarray3 = *arrobjpp
	var newmty *Metatype
	newmty = memdup3(mtype, sizeof(*mtype))
	newmty.Kind = Slice
	newmty.thisptr = data

	var valtyp *Metatype

	// TODO cxarray3 need kind field
	valtyp = metatype_bykind(Voidptr)
	if valtyp == nil {
	}
	assert(valtyp != nil)
	newmty.elemty = valtyp

	efc := Eface_new(newmty, data)
	return efc
}

func metatype_bykind(kind int) *Metatype {
	var mty *Metatype
	switch kind {
	case Int:
		mty = &int_metatype // from C
	case String:
		mty = &string_metatype
	case Float32:
		mty = &float32_metatype
	case Voidptr:
		mty = &voidptr_metatype
	case Struct:
	}
	return mty
}

type MethodObject struct {
	Ptr  voidptr // func pointer
	This voidptr
}

type wideptr struct {
	Ptr voidptr
	Obj voidptr
}

type typealg struct {
	hash  func(voidptr, int) usize
	equal func(voidptr, voidptr) bool
}

// Type is here for the purposes of documentation only. It is a stand-in
// for any Go type, but represents the same type for any given function
// invocation.
type Type int
type Type1 int
type IntegerType int
type FloatType float32

func typeof(val voidptr) *Metatype
func valueof(val voidptr) voidptr

func typeof_goimpl(tyobjx voidptr) *Metatype {
	var tyobj *Metatype
	tyobj = (*Metatype)(tyobjx)
	return tyobj
}

func (mty *Metatype) Name() string {
	return gostring(mty.Str)
}

func (mty *Metatype) KindName() string {
	kind := mty.Kind

	switch kind {
	case Struct:
		return "struct"
	case Map:
		return "map"
	case Array:
		return "array"
	case Slice:
		return "slice"
	case Chan:
		return "chan"
	case Voidptr:
		return "voidptr"
	case Byteptr:
		return "byteptr"
	case Charptr:
		return "charptr"
	case Wideptr:
		return "wideptr"
	}

	if kind >= Invalid && kind <= UnsafePointer {
		return gostring(mty.Str)
	}
	return "unktykind"
}

func (mty *Metatype) sizeof() int  { return mty.Size }
func (mty *Metatype) alignof() int { return mty.Align }

// all
func (mty *Metatype) New() voidptr {
	return nil
}

// map
func (mty *Metatype) KeySize() int {
	return mty.Align
}

// map
func (mty *Metatype) Key() *Metatype {
	return mty.keyty
}

// map/slice/array/ptr
func (mty *Metatype) ElemSize() int {
	return mty.Align
}

// map/slice/array/ptr
func (mty *Metatype) Elem() *Metatype {
	return mty.elemty
}

// map/slice/array
func (mty *Metatype) Len() int {
	return mty.Align
}

// map/slice/array
func (mty *Metatype) Cap() int {
	return mty.Align
}

// struct
func (mty *Metatype) NumField() int {
	return mty.count1
}

// struct
func (mty *Metatype) NumMethod() int {
	return mty.count2
}

// struct
func (mty *Metatype) Field(i int) *StructField {
	fldcnt := mty.NumField()
	fldname := mty.FieldName(i)
	fldty := mty.FieldType(i)

	fldo := &StructField{}
	fldo.Name = fldname
	fldo.Type = fldty
	fldo.Index = i
	fldo.Offset = 0

	return fldo
}

// struct
// just temporary
func (mty *Metatype) FieldName(i int) string {
	var ptrpp *byteptr = mty.extptr
	cstr := ptrpp[i]
	return gostring(cstr)
}

// struct
// just temporary
func (mty *Metatype) FieldType(i int) *Metatype {
	fldcnt := mty.NumField()
	var ptrpp *byteptr = mty.extptr
	var fldty *Metatype
	fldty = ptrpp[fldcnt+i]
	return fldty
}

// struct
func (mty *Metatype) FieldByName(name string) *StructField {
	fldcnt := mty.NumField()
	var ptrpp *byteptr = mty.extptr
	for i := 0; i < fldcnt; i++ {
		ptr := ptrpp[i]
		fldname := gostring(ptr)
		if fldname == name {
			return mty.Field(i)
		}
	}
	return nil
}

// struct
func (mty *Metatype) Method(i int) *Method {
	mtho := &Method{}
	return mtho
}

// struct
func (mty *Metatype) MethodByName(name string) *Method {
	mtho := &Method{}
	return mtho
}

// func
func (mty *Metatype) IsVariadict() bool {
	return false
}

// func
func (mty *Metatype) NumIn() int {
	return 0
}

// func
func (mty *Metatype) NumOut() int {
	return 0
}

// func
func (mty *Metatype) In(i int) *Metatype {
	return nil
}

// func
func (mty *Metatype) Out(i int) *Metatype {
	return nil
}

type ifctab struct {
	inter *interfacetype
	Type  *Metatype
	hash  uint32
	pad4  uint32
	fun   usize // [1]uintptr
}
type Iface struct {
	itab *ifctab
	Data voidptr
}

func (ifc *Iface) Empty() bool {
	return true
}

func (ifc *Iface) NumMethod() int {
	return 0
}

func (ifc *Iface) NumEmbeddeds() int {
	return 0
}
