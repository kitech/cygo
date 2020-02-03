package dwarf

/*
 */
import "C"

type Dwarf struct {
	filename string
	dbg      Debug
	err      Error

	sfilesv []string
	sfilesm map[Die][]int // => index list in sfilesv
}

func NewDwarf() *Dwarf {
	dwr := &Dwarf{}
	return dwr
}

func (dwr *Dwarf) Open(filename string) bool {
	dwr.filename = filename
	truepath, dbg, dwerr := init_path(filename)
	dwr.dbg = dbg
	dwr.err = dwerr
	return dwerr.Okay()
}

func (dwr *Dwarf) Close() bool {
	dwerr := finish(dwr.dbg)
	return dwerr.Okay()
}

func (dwr *Dwarf) Version() string {
	return package_version()
}

type CUItem struct {
	Name     string
	File     string
	Index    int
	Lang     string
	Content  string
	Producer string
	Dies     []*DieItem
}
type DieItem struct {
	Attrs []*AttrItem
}
type AttrItem struct {
	Kind  int
	Form  int
	Value interface{}
}

type CUHeaderb struct {
	Length    Unsigned
	Verstamp  Half
	Abbrevoff Off
	Addrsize  Half
	NextOff   Unsigned
	Dwerr     Error
}

// CU = compileunit
// Die = dwarf info entry?

func (dwr *Dwarf) PrintCUList() {
	cucnt := 0
	for idx := 0; ; idx++ {
		cuhdr, err1 := next_cu_header4(dwr.dbg)
		if err1.Fail() {
			println("heheh", err1.Error())
			break
		}
		cucnt++

		var nodie Die
		cudie, dwerr := siblingof(dwr.dbg, nodie)
		println(dwerr, cudie)
		if dwerr.Fail() {
			println("hehe", dwerr.Error())
		}

		var cusfiles []string
		{
			// 只有 cu 能够 srcfiles()
			sfiles, dwerr := srcfiles(cudie)
			println("sfiles", sfiles.len(), dwerr)
			for idx, sfile := range sfiles {
				println("sfiles", idx, sfile)
			}
			cusfiles = sfiles
		}

		dwr.get_die_and_siblings(cudie, 0, cusfiles)
		dealloc(dwr.dbg, cudie, DW_DLA_DIE)
	}
}

func (dwr *Dwarf) get_die_and_siblings(indie Die, inlvl int, cusfiles []string) {
	var subdie, curdie, sibdie Die
	var dwerr Error

	curdie = indie
	dwr.print_die_data(indie, inlvl, cusfiles)
	subdie, dwerr = child(curdie)
	if dwerr.Okay() {
		println(111, child, inlvl)
		dwr.get_die_and_siblings(subdie, inlvl+1, cusfiles)
		sibdie = subdie
		for cnter := 0; dwerr.Okay(); cnter++ {
			curdie = sibdie
			sibdie, dwerr = siblingof(dwr.dbg, curdie)
			println(222, curdie, sibdie, dwerr, cnter)
			dwr.get_die_and_siblings(sibdie, inlvl+1, cusfiles)
		}
	}
}

const ctrue = 1 // TODO move to builtin
const cfalse = 0

func (dwr *Dwarf) print_die_data(printme Die, lvl int, cusfiles []string) {
	sfiles := cusfiles
	var has_line_data bool
	{
		battr, dwerr := hasattr(printme, DW_AT_decl_line)
		has_line_data = dwerr.Okay() && battr == ctrue
		println("line", has_line_data, lvl, battr, dwerr, DW_AT_decl_line)
	}
	{
		battr, dwerr := hasattr(printme, DW_AT_decl_file)
		// has_line_data = ret == DW_DLV_OK && battr == ctrue
		var filename string
		if has_line_data {
			attr1, dwerr := attr(printme, DW_AT_decl_file)
			val, dwerr2 := formudata(attr1)
			println(val, sfiles.len())
			filename = sfiles[val-1]
		}
		println("file", has_line_data, lvl, battr, dwerr, DW_AT_decl_file, filename)
	}
	{
		battr, dwerr := hasattr(printme, DW_AT_location)
		// has_line_data = ret == DW_DLV_OK && battr == ctrue
		println("loc", has_line_data, lvl, battr, dwerr, DW_AT_location)
	}
	{
		diename, dwerr := diename(printme)
		println("name", diename, dwerr, dwerr, lvl)
		dealloc(dwr.dbg, diename, DW_DLA_STRING)
		dealloc(dwr.dbg, diename, DW_DLA_STRING)
	}
	{
		attrs, attrcnt, dwerr := attrlist(printme)
		println("attrs", attrcnt, dwerr)
		for i := 0; i < attrcnt; i++ {
			attr1 := attrs[i]
			var attrname string
			{
				attrno, dwerr := whatattr(attr1)
				attrname = AttrKindName(attrno)
				println("attr", i, attrno, dwerr, attrname)
				switch attrno {
				}
			}
			{
				formno, dwerr := whatform(attr1)
				println("form", i, formno, dwerr)
				switch formno {
				case DW_FORM_data2, DW_FORM_data1, DW_FORM_udata:
					val, dwerr := formudata(attr1)
					println("attr udata", i, formno, dwerr, val, attrname)
				case DW_FORM_addr:
					addr, dwerr := formaddr(attr1)
					println("attr addr", i, formno, dwerr, addr, attrname)
				case DW_FORM_strp:
					str, dwerr := formstring(attr1)
					println("attr strp", i, formno, dwerr, str)
				case DW_FORM_strx1:
					str, dwerr := formstring(attr1)
					println("attr strx1", i, formno, dwerr, str)
				}
			}
		}
	}
	println("haslineno", has_line_data, lvl)
	if has_line_data {

	}
}

var ErrNoEntry Error = -1

func packerror(ret int, dwerr Error) Error {
	if ret == DW_DLV_NO_ENTRY {
		return ErrNoEntry
	}
	if ret != DW_DLV_OK && dwerr == nil {
		return "wt" + ret.repr()
	}
	return dwerr
}

// func (dwerr Error) addr() *Error { return &dwerr } // not works
func (dwerr *Error) with(ret int) {
	dwerr2 := *dwerr
	dwerr3 := packerror(ret, dwerr2)
	if dwerr3 != dwerr2 {
		*dwerr = dwerr3
	}
}

func (dwerr Error) Okay() bool {
	return dwerr == nil
}
func (dwerr Error) Fail() bool {
	return dwerr != nil
}
func (dwerr Error) Errno() int {
	if dwerr == ErrNoEntry {
		return -1
	}
	rv := C.dwarf_errno(dwerr)
	return rv
}
func (dwerr Error) Errmsg() string {
	if dwerr == ErrNoEntry {
		return "NoEntry"
	}
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
