package xlog

// TODO compiler type/symbol order

type Dwarf struct {
	filename string
	dbg      DwarfDebug
	err      DwarfError
	ret      int
}

func NewDwarf() *Dwarf {
	dwr := &Dwarf{}
	return dwr
}

func (dwr *Dwarf) Open(filename string) bool {
	dwr.filename = filename
	truepath, dbg, dwerr, ret := dwarf_init_path(filename)
	return true
}
