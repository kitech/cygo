package xdl

/*
#include <ffi.h>

static void* cgo_ffi_type_void() {return (void*)&ffi_type_void;}
static void* cgo_ffi_type_uint8() {return (void*)&ffi_type_uint8;}
static void* cgo_ffi_type_sint8() {return (void*)&ffi_type_sint8;}
static void* cgo_ffi_type_uint16() {return (void*)&ffi_type_uint16;}
static void* cgo_ffi_type_sint16() {return (void*)&ffi_type_sint16;}
static void* cgo_ffi_type_uint32() {return (void*)&ffi_type_uint32;}
static void* cgo_ffi_type_sint32() {return (void*)&ffi_type_sint32;}
static void* cgo_ffi_type_uint64() {return (void*)&ffi_type_uint64;}
static void* cgo_ffi_type_sint64() {return (void*)&ffi_type_sint64;}
static void* cgo_ffi_type_float() {return (void*)&ffi_type_float;}
static void* cgo_ffi_type_double() {return (void*)&ffi_type_double;}
static void* cgo_ffi_type_pointer() {return (void*)&ffi_type_pointer;}
*/
import "C"

// flow:
// import symbol here
// c wrapper call these symbol
// asm call c wrapper
// go call asm

// //go:cgo_import_dynamic libc_printf printf  "libc.so.6"
// //go:cgo_import_dynamic libc_strlen strlen  "libc.so.6"

// //go:cgo_import_dynamic libffi_type_void ffi_type_void    "libffi.so"
// //go:cgo_import_dynamic libffi_type_uint8 ffi_type_uint8   "libffi.so"
// //go:cgo_import_dynamic libffi_type_sint8 ffi_type_sint8   "libffi.so"
// //go:cgo_import_dynamic libffi_type_uint16 ffi_type_uint16  "libffi.so"
// //go:cgo_import_dynamic libffi_type_sint16 ffi_type_sint16  "libffi.so"
// //go:cgo_import_dynamic libffi_type_uint32 ffi_type_uint32  "libffi.so"
// //go:cgo_import_dynamic libffi_type_sint32 ffi_type_sint32  "libffi.so"
// //go:cgo_import_dynamic libffi_type_uint64 ffi_type_uint64  "libffi.so"
// //go:cgo_import_dynamic libffi_type_sint64 ffi_type_sint64  "libffi.so"
// //go:cgo_import_dynamic libffi_type_float ffi_type_float   "libffi.so"
// //go:cgo_import_dynamic libffi_type_double ffi_type_double  "libffi.so"
// //go:cgo_import_dynamic libffi_type_pointer ffi_type_pointer "libffi.so"

// func goasm_ffi_type_void() voidptr
// func goasm_ffi_type_uint8() voidptr
// func goasm_ffi_type_sint8() voidptr
// func goasm_ffi_type_uint16() voidptr
// func goasm_ffi_type_sint16() voidptr
// func goasm_ffi_type_uint32() voidptr
// func goasm_ffi_type_sint32() voidptr
// func goasm_ffi_type_uint64() voidptr
// func goasm_ffi_type_sint64() voidptr
// func goasm_ffi_type_float() voidptr
// func goasm_ffi_type_double() voidptr
// func goasm_ffi_type_pointer() voidptr

// error: mkuse/go-ffi._Cvar_ffi_type_void: relocation target ffi_type_void not defined for ABI0 (but is defined for ABI0)
// var a = voidptr(&C.ffi_type_void)

var FFITypeVoid voidptr
var FFITypeUint8 voidptr
var FFITypeSint8 voidptr
var FFITypeUint16 voidptr
var FFITypeSint16 voidptr
var FFITypeUint32 voidptr
var FFITypeSint32 voidptr
var FFITypeUint64 voidptr
var FFITypeSint64 voidptr
var FFITypeFloat voidptr
var FFITypeDouble voidptr
var FFITypePointer voidptr

func init() {
	FFITypeVoid = C.cgo_ffi_type_void()
	FFITypeUint8 = C.cgo_ffi_type_uint8()
	FFITypeSint8 = C.cgo_ffi_type_sint8()
	FFITypeUint16 = C.cgo_ffi_type_uint16()
	FFITypeSint16 = C.cgo_ffi_type_sint16()
	FFITypeUint32 = C.cgo_ffi_type_uint32()
	FFITypeSint32 = C.cgo_ffi_type_sint32()
	FFITypeUint64 = C.cgo_ffi_type_uint64()
	FFITypeSint64 = C.cgo_ffi_type_sint64()
	FFITypeFloat = C.cgo_ffi_type_float()
	FFITypeDouble = C.cgo_ffi_type_double()
	// 	FFITypeVoid = goasm_ffi_type_void()
	// 	FFITypeUint8 = goasm_ffi_type_uint8()
	// 	FFITypeSint8 = goasm_ffi_type_sint8()
	// 	FFITypeUint16 = goasm_ffi_type_uint16()
	// 	FFITypeSint16 = goasm_ffi_type_sint16()
	// 	FFITypeUint32 = goasm_ffi_type_uint32()
	// 	FFITypeSint32 = goasm_ffi_type_sint32()
	// 	FFITypeUint64 = goasm_ffi_type_uint64()
	// 	FFITypeSint64 = goasm_ffi_type_sint64()
	// 	FFITypeFloat = goasm_ffi_type_float()
	// 	FFITypeDouble = goasm_ffi_type_double()
	// 	FFITypePointer = goasm_ffi_type_pointer()
	FFITypePointer = C.cgo_ffi_type_pointer()
}

//go:cgo_import_dynamic libffi_prep_cif ffi_prep_cif  "libffi.so"
//go:cgo_import_dynamic libffi_prep_cif_var ffi_prep_cif_var  "libffi.so"
//go:cgo_import_dynamic libffi_call ffi_call  "libffi.so"

func goasm_ffi_prep_cif(voidptr, uint32, uint32, voidptr, voidptr) uint32
func goasm_ffi_prep_cif_var(voidptr, uint32, uint32, uint32, voidptr, voidptr) uint32
func goasm_ffi_call(voidptr, voidptr, voidptr, voidptr)
func goasm_ffi_default_abi() uint32

type Cif struct {
	abi      uint32
	nargs    uint32
	argtypes voidptr
	rtype    voidptr
	nbytes   uint32
	flags    uint32

	retval uint64
	argvec []voidptr
}

func (cif *Cif) cptr() voidptr { return voidptr(&cif.abi) }
func (cif *Cif) rptr() voidptr { return voidptr(&cif.retval) }
func (cif *Cif) vptr() voidptr {
	if len(cif.argvec) > 0 {
		return voidptr(&cif.argvec[0])
	}
	return nil
}

func PrepCif(retype int, argtys []int) *Cif {
	var cif = &Cif{}
	cifc := cif.cptr()
	retypo := ity2pty(retype)
	argtypos := ity2ptys(argtys)
	var ap voidptr
	if len(argtypos) > 0 {
		ap = voidptr(&argtypos[0])
		cif.argvec = make([]voidptr, len(argtys))
	}

	r := goasm_ffi_prep_cif(cifc, goasm_ffi_default_abi(), uint32(len(argtys)), retypo, ap)
	// log.Printf("r=%d,cif=%+v\n", r, cif)
	if int(r) != FFI_OK {
	}
	return cif
}
func PrepCifVar() *Cif {
	var cif = &Cif{}
	// println(FFITypeVoid, unsafe.Sizeof(cif))
	println(FFITypeVoid, sizeof(cif))
	cifc := cif.cptr()
	r := goasm_ffi_prep_cif_var(cifc, goasm_ffi_default_abi(), 0, 0, FFITypeVoid, cifc)
	// log.Printf("r=%d,cif=%+v\n", r, cif)
	return cif
}
func Call(cif *Cif, fnptr voidptr, argvals []interface{}) interface{} {
	// 这些局部变量影响速度，否则还是可以与cgo比较的
	var arg0val int = 5
	cif.argvec[0] = voidptr(&arg0val)
	if len(argvals) == 2 {
		cif.argvec[1] = voidptr(&arg0val)
	}

	goasm_ffi_call(cif.cptr(), fnptr, cif.rptr(), cif.vptr())
	return cif.retval
}

const cifsz = 32
const (
	FFI_OK          = int(C.FFI_OK)
	FFI_BAD_TYPEDEF = int(C.FFI_BAD_TYPEDEF)
	FFI_BAD_ABI     = int(C.FFI_BAD_ABI)
)

const (
	/* If these change, update src/mips/ffitarget.h. */
	TYPE_VOID       = int(C.FFI_TYPE_VOID)
	TYPE_INT        = int(C.FFI_TYPE_INT)
	TYPE_FLOAT      = int(C.FFI_TYPE_FLOAT)
	TYPE_DOUBLE     = int(C.FFI_TYPE_DOUBLE)
	TYPE_LONGDOUBLE = int(C.FFI_TYPE_LONGDOUBLE)
	TYPE_UINT8      = int(C.FFI_TYPE_UINT8)
	TYPE_SINT8      = int(C.FFI_TYPE_SINT8)
	TYPE_UINT16     = int(C.FFI_TYPE_UINT16)
	TYPE_SINT16     = int(C.FFI_TYPE_SINT16)
	TYPE_UINT32     = int(C.FFI_TYPE_UINT32)
	TYPE_SINT32     = int(C.FFI_TYPE_SINT32)
	TYPE_UINT64     = int(C.FFI_TYPE_UINT64)
	TYPE_SINT64     = int(C.FFI_TYPE_SINT64)
	TYPE_STRUCT     = int(C.FFI_TYPE_STRUCT)
	TYPE_POINTER    = int(C.FFI_TYPE_POINTER)
	TYPE_COMPLEX    = int(C.FFI_TYPE_COMPLEX)
)

func ity2ptys(tys []int) (ptys []voidptr) {
	for _, ty := range tys {
		ptys = append(ptys, ity2pty(ty))
	}
	return
}

func ity2pty(ty int) voidptr {
	switch ty {
	case TYPE_VOID:
		return FFITypeVoid
	case TYPE_INT:
		return FFITypeSint32
	case TYPE_FLOAT:
		return FFITypeFloat
	case TYPE_DOUBLE:
		return FFITypeDouble
	case TYPE_LONGDOUBLE:
		return FFITypeDouble
	case TYPE_UINT8:
		return FFITypeUint8
	case TYPE_SINT8:
		return FFITypeSint8
	case TYPE_UINT16:
		return FFITypeUint16
	case TYPE_SINT16:
		return FFITypeSint16
	case TYPE_UINT32:
		return FFITypeUint32
	case TYPE_SINT32:
		return FFITypeSint32
	case TYPE_UINT64:
		return FFITypeUint64
	case TYPE_SINT64:
		return FFITypeSint64
	case TYPE_STRUCT:
		return FFITypePointer
	case TYPE_POINTER:
		return FFITypePointer
	case TYPE_COMPLEX:
		return FFITypePointer
	}
	return FFITypeVoid
}

func Keepme() {
	var a int
	_ = voidptr(nil)
}

func init() {
	println("ffi const:", FFI_OK, FFI_BAD_TYPEDEF, FFI_BAD_ABI)
	println("ffi dftabi:", goasm_ffi_default_abi())
	// force keep
	// rn := rand.Uint32() + 1
	rn := 1
	if rn == 0 {
		println("ffi", PrepCif, PrepCifVar, Call)
		println("ffi",
			FFITypeVoid,
			FFITypeUint8,
			FFITypeSint8,
			FFITypeUint16,
			FFITypeSint16,
			FFITypeUint32,
			FFITypeSint32,
			FFITypeUint64,
			FFITypeSint64,
			FFITypeFloat,
			FFITypeDouble,
			FFITypePointer,
		)
	}
}
