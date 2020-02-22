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
	String
	UnsafePointer

	// types for untyped values
	UntypedBool
	UntypedInt
	UntypedRune
	UntypedFloat
	UntypedComplex
	UntypedString
	UntypedNil
)

const (
	// endofOrignalGoType = iota + UntypedNil // TODO compiler
	endofOrignalGoType = iota + 25
	Voidptr
	Byteptr
	Charptr
)

const (
	Struct = 25
	Slice
	Array
	Map
	Ptr
	Chan
	Func
	// Interface
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
	Str        byteptr
	PtrToThis  voidptr
	// int32   // nameOff // string form
	// int32   // typeOff // type for pointer to this type, may be zero
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

// cxeface* cxrt_type2eface(voidptr _type, voidptr data);

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

type Wideptr struct {
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
	if kind >= Invalid && kind <= UnsafePointer {
		return gostring(mty.Str)
	}
	switch mty.Kind {
	case Struct:
		return "struct"
		// case Map:
		// return "map"
		// case Array:
		// return "array"
		// case Slice:
		// return "slice"
		// case Chan:
		// return "chan"
	}
	return "unktykind"
}

func (mty *Metatype) sizeof() int  { return mty.Size }
func (mty *Metatype) alignof() int { return mty.Align }

type Interface struct {
	todo int
}

func (ifc *Interface) Empty() bool {
	return true
}

func (ifc *Interface) NumMethods() int {
	return 0
}

func (ifc *Interface) NumEmbeddeds() int {
	return 0
}
