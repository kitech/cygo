package dwarf

/*
 */
import "C"

type CU voidptr
type _Die struct {
	die_parent    Die
	die_child     Die
	die_left      Die
	die_right     Die
	die_offset    uint64
	die_next_off  uint64
	die_abnum     uint64
	die_ab        Abbrev
	die_tag       Tag
	die_dbg       Debug
	die_cu        CU
	die_name      byteptr
	die_attrarray *Attribute
	// STAILQ_HEAD(, _Dwarf_Attribute)	die_attr; /* List of attributes. */
	// STAILQ_ENTRY(_Dwarf_Die) die_pro_next; /* Next die in pro-die list. */
}

// TODO incorrect result, maybe _Die struct not correct
func (die Die) Dbg() Debug {
	var undie *_Die = die
	return undie.die_dbg
}

func (dwerr Error) Errno() int {
	rv := C.dwarf_errno(dwerr)
	return rv
}

func (dwerr Error) Errmsg() string {
	rv := C.dwarf_errmsg(dwerr)
	emsg := gostring(rv)
	return emsg
}

func (dwerr Error) Error() string {
	if dwerr != nil {
		eno := dwerr.Errno()
		emsg := dwerr.Errmsg()
		emsg2 := "DWE " + eno.repr() + " " + emsg
		return emsg2
	}
	return ""
}
