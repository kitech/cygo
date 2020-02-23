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
	Fieldalign uint8    // alignment of struct field with this type
	Kind       uint8    // enumeration for C
	Alg        *typealg // *typeAlg // algorithm table
	Gcdata     byteptr  // garbage collection data
	Str        byteptr  // nameOff // string form
	PtrToThis  voidptr  // typeOff // type for pointer to this type, may be zero
	count1     uint8    // for uncommon type, like map/slice/ptr
	count2     uint8
	extptr     *voidptr
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

//export cxrt_type2eface
func type2eface(mtype voidptr, data voidptr) *Eface {
	var mty *Metatype = mtype
	efc := &Eface{}
	efc.Type = mtype
	efc.Data = data
	efc.Data = memdup3(data, mty.Size)
	return efc
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

func (ifc *Iface) NumMethods() int {
	return 0
}

func (ifc *Iface) NumEmbeddeds() int {
	return 0
}
