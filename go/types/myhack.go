package types

import "log"

// too late
func HackExtraBuiltin() {
	if true {
		return
	}
	tybno := UntypedNil
	tybinfo := IsUntyped
	{
		vptrty := &Basic{tybno << 1, tybinfo << 1, "voidptr"}
		Typ = append(Typ, vptrty)
	}
	{
		vptrty := &Basic{tybno << 2, tybinfo << 2, "byteptr"}
		Typ = append(Typ, vptrty)
	}
	log.Println("222222222")
}

// Nil represents the predeclared value nil.
type Nilptr struct {
	object
}
