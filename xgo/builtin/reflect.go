package builtin

type SliceHeader struct {
	Data uintptr
	Len  int
	Cap  int
}

type StringHeader struct {
	Data uintptr
	Len  int
}

type Method struct {
	// Name is the method name.
	// PkgPath is the package path that qualifies a lower case (unexported)
	// method name. It is empty for upper case (exported) method names.
	// The combination of PkgPath and Name uniquely identifies a method
	// in a method set.
	// See https://golang.org/ref/spec#Uniqueness_of_identifiers
	Name    string
	PkgPath string

	Type *Metatype // method type
	// Func  Value // func with receiver as first argument
	Index int // index for Type.Method
}

type StructField struct {
	Name      string
	PkgPath   string
	Type      *Metatype
	Tag       string  // StructTag
	Offset    uintptr // offset within struct, in bytes
	Index     int     // []int???
	Anonymous bool
}

type StructTag string

func (tag StructTag) Get() string {
	var s string
	s = tag
	return s
}

func (tag StructTag) Lookup(key string) (value string, ok bool) {
	return "", false
}

func ArrayOf(count int, elem *Metatype) *Metatype {
	return nil
}

func SliceOf(count int, elem *Metatype) *Metatype {
	return nil
}

func ChanOf(dir int, elem *Metatype) *Metatype {
	return nil
}

func MapOf(key *Metatype, elem *Metatype) *Metatype {
	return nil
}

// func FuncOf(in []*Metatype, out []*Metatype, variadict bool) *Metatype {
// 	return nil
// }

func PtrTo(elem *Metatype) *Metatype {
	p2typ := &Metatype{}
	p2typ.Kind = Ptr
	p2typ.elemty = elem
	p2typ.Size = sizeof(p2typ)

	return p2typ
}

func StructOf(fields []*StructField) *Metatype {
	return nil
}

/////
type Value struct {
	typ  *Metatype // *rtype???
	ptr  voidptr
	flag uintptr // flag type???
}

func (v *Value) Addr() *Value {
	return v.ptr
}
func (v *Value) CanAddr() bool {
	return true
}
func (v *Value) CanSet() bool {
	return false
}
func (v *Value) Pointer() uintptr {
	return v.ptr
}

func (v *Value) Bool() bool {
	typ := v.typ
	var rv bool
	switch typ.Kind {
	case Bool:
		var tv *bool = v.ptr
		rv = *tv
	}

	return rv
}

func (v *Value) Bytes() []byte {
	return nil
}

// in is the keyword now

func (v *Value) Call(args []*Value) []*Value {
	return nil
}
func (v *Value) CallSlice(args []*Value) []*Value {
	return nil
}

func (v *Value) Close() {
	return
}

func (v *Value) Cap() int {
	typ := v.typ
	switch typ.Kind {
	case Slice, Array:
		var arr *cxarray3 = v.ptr
		return arr.cap
	case Map:
		var ht *mirmap = v.ptr
		return ht.cap_
	default:
		return typ.Size
	}
	return 0
}

func (v *Value) Len() int {
	typ := v.typ
	switch typ.Kind {
	case Slice, Array:
		var arr *cxarray3 = v.ptr
		return arr.len
	case Map:
		var ht *mirmap = v.ptr
		return ht.len_
	default:
		return typ.Size
	}
	return 0
}

// eface/ptr
func (v *Value) Elem() *Value {
	return nil
}

func (v *Value) Float() float64 {
	typ := v.typ
	var rv float64
	switch typ.Kind {
	case Float32:
		var tv *float32 = v.ptr
		rv = *tv
	case Float64:
		var tv *float64 = v.ptr
		rv = *tv
	}

	return rv
}

// slice/array/string
func (v *Value) Index(i int) *Value {
	typ := v.typ
	switch typ.Kind {
	case Slice, Array:
		var arr *cxarray3 = v.ptr
		elemval := arr.get(i)
		elemty := metatype_bykind(typ.elemty.Kind)
		rv := &Value{}
		rv.typ = elemty
		rv.ptr = elemval
		return rv
	case String:
		var str *cxstring3 = v.ptr
		ch := str.ptr[i]
		var ih int = ch
		rv := &Value{}
		rv.typ = metatype_bykind(Uint8)
		rv.ptr = memdup3(&rv, sizeof(rv))
		return rv
	}
	return nil
}

func (v *Value) CanInterface() bool {
	return true
}

func (v *Value) ToInterface() interface{} {
	efc := Eface_new(v.typ, v.ptr)
	return efc
}

func (v *Value) IsNil() bool {
	return v.ptr == nil
}
func (v *Value) IsValid() bool {
	return v.typ.Kind > Invalid
}
func (v *Value) IsZero() bool {
	var rv bool
	typ := v.typ
	switch typ.Kind {
	case Int, Uint:
		var tv *int = v.ptr
		rv = *tv == 0
	case Int64, Uint64:
		var tv *int64 = v.ptr
		rv = *tv == 0
	case Int32, Uint32:
		var tv *int32 = v.ptr
		rv = *tv == 0
	case Int16, Uint16:
		var tv *int64 = v.ptr
		rv = *tv == 0
	}

	return rv
}
func (v *Value) Kind() int {
	return v.typ.Kind
}

func (v *Value) MapIndex(key *Value) *Value {
	return nil
}

func (v *Value) MapKeys() []*Value {
	return nil
}

func (v *Value) NumMethod() int {
	return v.typ.NumMethod()
}

func (v *Value) Method(i int) *Value {
	return nil
}

func (v *Value) MethodByName(name string) *Value {
	return nil
}

func (v *Value) NumField() int {
	return v.typ.NumField()
}

func (v *Value) Field(i int) *Value {
	return nil
}

func (v *Value) FieldByName(name string) *Value {
	return nil
}

func (v *Value) GetType() *Metatype {
	return v.typ
}

func (v *Value) Convert(typ *Metatype) *Value {
	return nil
}

func (v *Value) Int() int64 {
	typ := v.typ
	var rv int64
	switch typ.Kind {
	case Int:
		var tv *int = v.ptr
		rv = *tv
	case Int64:
		var tv *int64 = v.ptr
		rv = *tv
	case Int32:
		var tv *int32 = v.ptr
		rv = *tv
	case Int16:
		var tv *int64 = v.ptr
		rv = *tv
	}

	return rv
}

func (v *Value) Uint() uint64 {
	typ := v.typ
	var rv uint64
	switch typ.Kind {
	case Uint:
		var tv *uint = v.ptr
		rv = *tv
	case Int64:
		var tv *uint64 = v.ptr
		rv = *tv
	case Int32:
		var tv *uint32 = v.ptr
		rv = *tv
	case Int16:
		var tv *uint64 = v.ptr
		rv = *tv
	}

	return rv
}
func (v *Value) String() string {
	var rv string
	if v.typ.Kind == Invalid {
		return "<invalid Value>"
	} else if v.typ.Kind == String {
		var spp **cxstring3 = v.ptr
		rv = *spp
	} else {
		return "<" + gostring(v.typ.Str) + " Value>"
	}
	return rv
}

func ValueOf(iv interface{}) *Value {
	var ifcpp **Eface = &iv
	var ifc *Eface = *ifcpp
	val := &Value{}
	val.typ = ifc.Type
	val.ptr = ifc.Data

	return val
}

// PtrTo'd
func NewOf(typ *Metatype) *Value {
	rv := &Value{}
	rv.typ = PtrTo(typ)
	rv.ptr = malloc3(typ.Size)
	return rv
}

func NewAt(typ *Metatype, p voidptr) *Value {
	rv := &Value{}
	rv.typ = typ
	rv.ptr = p
	return rv
}

func Zero(typ *Metatype) *Value {
	rv := &Value{}
	rv.typ = typ
	rv.ptr = malloc3(typ.Size)
	return rv
}

func MakeSlice(typ *Metatype, len int, cap int) *Value {
	return nil
}

func MakeChan(typ *Metatype, buffer int) *Value {
	return nil
}

func MakeMap(typ *Metatype) *Value {
	return nil
}
func MakeMapWithSize(typ *Metatype, n int) *Value {
	return nil
}
