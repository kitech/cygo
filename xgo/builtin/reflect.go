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
	return nil
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
	return nil
}
func (v *Value) CanAddr() bool {
	return true
}
func (v *Value) CanSet() bool {
	return true
}
func (v *Value) Pointer() uintptr {
	return nil
}

func (v *Value) Bool() bool {
	return false
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
	return 0
}

func (v *Value) Len() int {
	return 0
}

func (v *Value) Elem() *Value {
	return nil
}

func (v *Value) Float() float64 {
	return 0
}

func (v *Value) Index(i int) *Value {
	return nil
}

func (v *Value) Int() int {
	return 0
}
func (v *Value) CanInterface() bool {
	return true
}

func (v *Value) ToInterface() interface{} {
	return nil
}

func (v *Value) IsNil() bool {
	return true
}
func (v *Value) IsValid() bool {
	return true
}
func (v *Value) IsZero() bool {
	return true
}
func (v *Value) Kind() int {
	return 0
}

func (v *Value) MapIndex(key *Value) *Value {
	return nil
}

func (v *Value) MapKeys() []*Value {
	return nil
}

func (v *Value) NumMethod() int {
	return 0
}

func (v *Value) Method(i int) *Value {
	return nil
}

func (v *Value) MethodByName(name string) *Value {
	return nil
}

func (v *Value) NumField() int {
	return 0
}

func (v *Value) Field(i int) *Value {
	return nil
}

func (v *Value) FieldByName(name string) *Value {
	return nil
}

func (v *Value) GetType() *Metatype {
	return nil
}

func (v *Value) Convert(typ *Metatype) *Value {
	return nil
}

func (v *Value) ToInt() int64 {
	return 0
}

func (v *Value) Uint() uint64 {
	return 0
}

func NewOf(typ *Metatype) *Value {
	return nil
}
func NewAt(typ *Metatype, p voidptr) *Value {
	return nil
}

func ValueOf(i interface{}) *Value {
	return nil
}

func Zero(typ *Metatype) *Value {
	return nil
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
