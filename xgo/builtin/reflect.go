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
