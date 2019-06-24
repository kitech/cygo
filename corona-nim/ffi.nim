# {.passc:"-I/usr/lib/libffi-3.2.1/include/"}
{.passc: staticExec("pkg-config --cflags libffi") .}
{.passl:"-lffi"}
{.compile:"ffi1.c"}

proc ffi_get_default_abi() : cint {.importc.}
var FFI_DEFAULT_ABI = ffi_get_default_abi()

type
    ppffi_type = ptr ptr ffi_type
    pffi_type = ptr ffi_type
    ffi_type = object
        size*: int # size_t
        alignment*: uint16 # cushort
        typ*: uint16 # cushort
        elements*: ptr ptr ffi_type

var ffi_type_void {.importc.} : ffi_type
var ffi_type_uint8 {.importc.} : ffi_type
var ffi_type_sint8 {.importc.} : ffi_type
var ffi_type_uint16 {.importc.} : ffi_type
var ffi_type_sint16 {.importc.} : ffi_type
var ffi_type_uint32 {.importc.} : ffi_type
var ffi_type_sint32 {.importc.} : ffi_type
var ffi_type_uint64 {.importc.} : ffi_type
var ffi_type_sint64 {.importc.} : ffi_type
var ffi_type_float {.importc.} : ffi_type
var ffi_type_double {.importc.} : ffi_type
var ffi_type_pointer {.importc.} : ffi_type

type
    ffi_status = enum
        FFI_OK = 0
        FFI_BAD_TYPEDEF
        FFI_BAD_ABI

type
    pffi_cif = ptr ffi_cif
    ffi_cif = object
        abi*: cint
        nargs*: cuint
        arg_types*: pffi_type
        rtype*: pffi_type
        bytes*: cuint
        flags*: cuint

###
proc ffi_type_size() : cint {.importc.}
proc ffi_cif_size() : cint {.importc.}

assert(ffi_type_size() == sizeof(ffi_type), $ffi_type_size() & "=?" & $sizeof(ffi_type))
assert(ffi_cif_size() == sizeof(ffi_cif), $ffi_cif_size() & "=?" & $sizeof(ffi_cif))

proc ffi_prep_cif(cif:pffi_cif, abi:cint, nargs:cuint, rtype:pffi_type, atypes:pffi_type) : cint {.importc}
proc ffi_prep_cif(cif:pffi_cif, abi:cint, nargs:cuint, rtype:pffi_type, atypes:pointer) : cint {.importc}
proc ffi_prep_cif(cif:pffi_cif, abi:cint, nargs:uint, rtype:pffi_type, atypes:pointer) : cint {.importc}

proc ffi_call(cif:pffi_cif, fn:pointer, rvalue: pointer, avalue: pointer) {.importc.}
proc ffi_call(cif:pffi_cif, fn:proc, rvalue: pointer, avalue: pointer) {.importc.}
proc ffi_call(cif:pffi_cif, fn:proc, rvalue: pffi_type, avalue: pointer) {.importc.}

proc dump_pointer_array(n:int, p:pointer) {.importc.}
proc pointer_array_new(n:int) : pointer {.importc.}
proc pointer_array_set(p:pointer, idx:int, v:pointer) {.importc.}
proc pointer_array_get(p:pointer, idx:int) :pointer {.importc.}
proc pointer_array_free(p:pointer) {.importc.}
proc pointer_array_addr(p:pointer) :pointer {.importc.}
proc calloc(count:csize, size:csize) :pointer {.importc.}
proc free(p:pointer) {.importc.}

#[

ffi_status ffi_prep_cif(ffi_cif *cif,
			                  ffi_abi abi,
			                  unsigned int nargs,
			                  ffi_type *rtype,
			                  ffi_type **atypes);
void ffi_call(ffi_cif *cif,
	      void (*fn)(void),
	      void *rvalue,
	      void **avalue);
]#
#{.hint[XDeclaredButNotUsed]:off.}

