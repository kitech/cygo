package builtin

import "unsafe"

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
	endofOrignalGoType = iota + 25
	Voidptr
	Byteptr
	Charptr
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
	Size       uintptr // type size
	Ptrdata    uintptr // size of memory prefix holding all pointers
	Hash       uint32  // hash of type; avoids computation in hash tables
	Tflag      uint8   // tflag   // extra type information flags
	Align      uint8   // alignment of variable with this type
	Fieldalign uint8   // alignment of struct field with this type
	Kind       uint8   // enumeration for C
	Alg        voidptr // *typeAlg // algorithm table
	Gcdata     *byte   // garbage collection data
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

type MethodObject struct {
	Ptr  voidptr // func pointer
	This voidptr
}

type Wideptr struct {
	Ptr voidptr
	Obj voidptr
}
