package dwarf

// TODO compiler type/symbol order

type Dwarf struct {
	filename string
	dbg      Debug
	err      Error
	ret      int
}

func NewDwarf() *Dwarf {
	dwr := &Dwarf{}
	return dwr
}

func (dwr *Dwarf) Open(filename string) bool {
	dwr.filename = filename
	truepath, dbg, dwerr, ret := init_path(filename)
	dwr.dbg = dbg
	dwr.err = dwerr
	dwr.ret = ret
	return ret == DW_DLV_OK
}

func (dwr *Dwarf) Close() bool {
	dwerr, ret := finish(dwr.dbg)
	return ret == DW_DLV_OK
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
		cuhdr := next_cu_header4(dwr.dbg)
		if cuhdr.Ret == DW_DLV_ERROR {
			println("err", cuhdr.Ret)
			break
		}
		if cuhdr.Ret == DW_DLV_NO_ENTRY {
			println("done", cuhdr.Ret)
			break
		}
		cucnt++
		println(idx)

		var nodie Die
		cudie, dwerr, ret := siblingof(dwr.dbg, nodie)
		println(ret, cudie)
		if ret == DW_DLV_ERROR {
			println("err", ret, idx)
			break
		}
		if ret == DW_DLV_NO_ENTRY {
			println("no entry on CU die", idx)
			break
		}

		dwr.get_die_and_siblings(cudie, 0)
		dealloc(dwr.dbg, cudie, DW_DLA_DIE)
	}
}

func (dwr *Dwarf) get_die_and_siblings(indie Die, inlvl int) {
	var subdie, curdie, sibdie Die // TODO compiler
	var dwerr Error
	var ret int

	dwr.print_die_data(indie, inlvl)

	subdie, dwerr, ret = child(indie)
	if ret != DW_DLV_OK {
		println(111, ret, child, inlvl)
		dwr.get_die_and_siblings(subdie, inlvl+1)
		sibdie = subdie
		for cnter := 0; ret == DW_DLV_OK; cnter++ {
			curdie = sibdie
			sibdie, dwerr, ret = siblingof(dwr.dbg, curdie)
			// sibdie2, dwerr2, ret2 := siblingof(dwr.dbg, curdie)
			// sibdie = sibdie2
			// dwerr = dwerr2
			// ret = ret2
			println(222, ret, curdie, sibdie, dwerr, cnter)
			dwr.get_die_and_siblings(sibdie, inlvl+1)
		}
	}
}

const ctrue = 1 // TODO move to builtin
const cfalse = 0

func (dwr *Dwarf) print_die_data(printme Die, lvl int) {
	var has_line_data bool
	{
		battr, dwerr, ret := hasattr(printme, DW_AT_decl_line)
		has_line_data = ret == DW_DLV_OK && battr == ctrue
		println("line", has_line_data, lvl, ret, battr, dwerr, DW_AT_decl_line)
	}
	{
		battr, dwerr, ret := hasattr(printme, DW_AT_decl_file)
		// has_line_data = ret == DW_DLV_OK && battr == ctrue
		println("file", has_line_data, lvl, ret, battr, dwerr, DW_AT_decl_file)
	}
	{
		battr, dwerr, ret := hasattr(printme, DW_AT_location)
		// has_line_data = ret == DW_DLV_OK && battr == ctrue
		println("loc", has_line_data, lvl, ret, battr, dwerr, DW_AT_location)
	}
	{
		diename, dwerr, ret := diename(printme)
		println("name", diename, dwerr, ret, lvl)
		dealloc(dwr.dbg, diename, DW_DLA_STRING)
		dealloc(dwr.dbg, diename, DW_DLA_STRING)
	}
	{
		attrs, attrcnt, dwerr, ret := attrlist(printme)
		println("attrs", attrcnt, ret)
		for i := 0; i < attrcnt; i++ {
			attr1 := attrs[i]
			{
				attrno, dwerr, ret := whatattr(attr1)
				attrname := AttrKindName(attrno)
				println("attr", i, attrno, ret, attrname)
				switch attrno {
				}
			}
			{
				formno, dwerr, ret := whatform(attr1)
				println("form", i, formno, ret)
				// var formno2 int = formno
				switch formno {
				case DW_FORM_strp:
					str, dwerr, ret := formstring(attr1)
					println("attr strp", i, formno, ret, str)
				case DW_FORM_strx1:
					str, dwerr, ret := formstring(attr1)
					println("attr strx1", i, formno, ret, str)
				}
			}
		}
	}
	println(has_line_data, lvl)
	if has_line_data {

	}
}
