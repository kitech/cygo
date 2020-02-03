package dwarf

/*
void dwarf2_get_elf_start_addr(void** exeadr, void** etxtadr, void**mainadr)  {
    * exeadr = &__executable_start;
    * etxtadr = &__etext;
    // printf("0x%lx\n", (unsigned long)&__executable_start);
    // printf("0x%lx\n", (unsigned long)&__etext);
    extern int main(int argc, char**argv);
    * mainadr = (void*)main;
printf("%p %p\n", main, (void*)main-*exeadr);
}

*/
import "C"

var exeadr voidptr
var etxtadr voidptr
var mainadr voidptr

func init() {
	C.dwarf2_get_elf_start_addr(&exeadr, &etxtadr, &mainadr)
	// println("elf start addr", exeadr, etxtadr)
}

type Dwarf struct {
	filename string
	dbg      Debug
	err      Error

	sfilesv []string
	sfilesm map[Die][]int // => index list in sfilesv
	a2l     *Addr2Line
}

func NewDwarf() *Dwarf {
	dwr := &Dwarf{}
	// dwr.inita2l()
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

func (dwr *Dwarf) RangeCU(itfn func(idx int, cudie Die) bool) {

}
func (dwr *Dwarf) nopcu(idx int, cudie Die) bool {
	return true
}
func (dwr *Dwarf) testrcu1() {
	// dwr.RangeCU(dwr.nopcu) // TODO compiler
}
func (dwr *Dwarf) testrcu2() {
	// TODO compiler
	/*
		dwr.RangeCU(func(idx int, die Die) bool {
			return true
		})
	*/
}

type CUIter struct {
	dbg Debug
	idx int
	eof bool
	err Error
}

func (dw *Dwarf) NewCUIterm() *CUIter {
	cuit := &CUIter{}
	cuit.dbg = dw.dbg
	return cuit
}

func (cuit *CUIter) Next() (*CUHeader4, Error) {
	dbg := cuit.dbg
	cuhdr, err1 := next_cu_header4(dbg)
	if err1.Fail() {
		cuit.eof = true
		return nil, err1
	}
	cudie, dwerr := siblingof2(dbg, 0)
	if dwerr.Fail() {
		return nil, dwerr
	}
	cuhdr.CUdie = cudie
	cuhdr.Index = cuit.idx
	cuit.idx++
	return cuhdr, nil
}
func (cuit *CUIter) Index() int { return cuit.idx - 1 }

// sometimes need break iterate
func (cuit *CUIter) SkipTail() {
	for cuit.eof == false {
		cuit.Next()
	}
}

func (dwr *Dwarf) PrintCUList() {
	cuidx := 0
	for ; ; cuidx++ {
		cuhdr, err1 := next_cu_header4(dwr.dbg)
		if err1.Fail() {
			println("heheh", err1.Error())
			break
		}
		cudie, dwerr := siblingof2(dwr.dbg, 0)
		println(dwerr, cudie)
		if dwerr.Fail() {
			println("hehe", dwerr.Error())
			break
		}

		var cusfiles []string
		{
			// 只有 cu 能够 srcfiles()
			sfiles, dwerr := srcfiles(cudie)
			println("sfiles", sfiles.len(), dwerr)
			for idx, sfile := range sfiles {
				println("sfiles", cuidx, sfile)
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
			dwr.dealloc(attr1, DW_DLA_ATTR)
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
func (dw *Dwarf) dealloc(space voidptr, type_ int) {
	dealloc(dw.dbg, space, type_)
}

///
type Addr2Line struct {
	cucnt int
	minpc Addr
	maxpc Addr

	pclntab []Line
	ctxts   []LineContext

	cufilesv []string      // global uniq filenames
	cufilesm map[int][]int // cuidx => file index list of cufilesv
}

func newAddr2Line() *Addr2Line {
	a2l := &Addr2Line{}
	a2l.minpc = DW_DLV_BADADDR
	return a2l
}

func (a2l *Addr2Line) setminpc(pc Addr) {
	if pc < a2l.minpc {
		a2l.minpc = pc
	}
}
func (a2l *Addr2Line) setmaxpc(pc Addr) {
	if pc > a2l.maxpc {
		a2l.maxpc = pc
	}
}

func (dw *Dwarf) inita2l() {
	a2l := newAddr2Line()
	dw.a2l = a2l
	dw.calc_minmax_pc()
	tabsz := usize(a2l.maxpc - a2l.minpc)
	println("tabsz", tabsz, a2l.cucnt)
	a2l.pclntab = make([]Line, tabsz)
	a2l.ctxts = make([]LineContext, a2l.cucnt)
	dw.calc_lookup_table()
	dw.check_lookup_table()
	dw.calc_cufiles()
	dw.check_cufiles()
}

func (dw *Dwarf) Addr2Line(addrx voidptr) (
	filename string, fileline int, found bool) {
	addr := dw.addr2virt(addrx)
	filename, fileline, found = dw.getfileline(addr)
	if filename.len() == 0 {
		filename = "???"
	}
	return
}

func (dw *Dwarf) addr2virt(addrx voidptr) Addr {
	// TODO
	a2l := dw.a2l

	var magic Addr = a2l.minpc - 40 // a2l.minpc // readelf -e ./exename
	var addr Addr = 0x5eeee
	addr = addrx - exeadr
	//	addr = addr + magic - 0x8
	// println("adr2v", addrx, "=>", addr, exeadr, mainadr)
	// println("minpc", a2l.minpc, "maxpc", a2l.maxpc)
	if false {
		addr = addrx - etxtadr
		addr = addr + magic - 0x8
		println("adr2v", addrx, "=>", addr, etxtadr)
	}
	return addr
}

// only receive short/virt addr
func (dw *Dwarf) getfileline(addrx voidptr) (
	filename string, fileline int, found bool) {
	a2l := dw.a2l
	var addr Addr = addrx
	cuit := dw.NewCUIterm()
	for stop := false; !stop; {
		cuhdr, err1 := cuit.Next()
		if err1.Fail() {
			break
		}
		cudie := cuhdr.CUdie
		cuidx := cuhdr.Index

		indie := dw.pc_in_die(cudie, addr)
		// println(cuidx, indie, cudie, addr)
		if indie {
			rfileno, rlineno, lookres := dw.lookup_pc_cu(cudie, addr)
			cuit.SkipTail()
			if lookres {
				// filename, fileline = dw.print_pcline2(retline, addr, cuidx)
				filename = dw.fileno2file(cuidx, rfileno)
				fileline = rlineno
			}
			found = lookres
			stop = true
		}
		dw.dealloc(cudie, DW_DLA_DIE)
	}
	if !found {
		println("not found", cuit.idx, addr)
	}
	return
}
func (dw *Dwarf) pc_in_die(die Die, pc Addr) (found bool) {
	culowpc, err1 := lowpc(die)
	if err1.Okay() {
		if pc == culowpc {
			return true
		}
		cuhighpc, err2 := highpcx(die)
		if err2.Okay() {
			if pc >= culowpc && pc < cuhighpc {
				return true
			}
		}
		// println("111", culowpc, cuhighpc, pc)
	}

	attr1, err2 := attr(die, DW_AT_ranges)
	if err2.Okay() {
		offset, err := global_formref(attr1)
		if err.Okay() {
			var baseaddr Addr
			ranges, listlen, bytecnt, err := get_ranges1(dw.dbg, offset, die)
			for i := 0; i < listlen; i++ {
				// cur := ranges + i // TODO compiler
				var cur *Ranges
				cur = usize(ranges) + usize(i*sizeof(Ranges))
				if cur.dwr_type == DW_RANGES_ENTRY {
					rglowpc := baseaddr + cur.dwr_addr1
					rghighpc := baseaddr + cur.dwr_addr2
					if pc >= rglowpc && pc < rghighpc {
						found = true
						break
					}
				} else if cur.dwr_type == DW_RANGES_ADDRESS_SELECTION {
					baseaddr = cur.dwr_addr2
				} else {
					baseaddr = culowpc
				}
			}
			ranges_dealloc(dw.dbg, ranges, listlen)
		}
		dw.dealloc(attr1, DW_DLA_ATTR)
	}

	return
}
func (dw *Dwarf) lookup_pc_cu(die Die, pc Addr) (
	retfileno int, retlineno int, found bool) {
	verout, tabcnt, linectx, err1 := srclines2(die)
	if err1 == ErrNoEntry {
		return
	}
	// defer srclines_dealloc2(linectx) // TODO compiler

	for tabcnt == 1 {
		linebuf, linecnt, err1 := srclines_from_linecontext(linectx)
		if err1.Fail() {
			break
		}
		var prev_lineaddr Addr
		var prev_line Line
		for i := 0; i < linecnt; i++ {
			line := linebuf[i]
			lnaddr, err1 := lineaddr(line)
			if pc == lnaddr {
				last_pc_line := line
				for j := i + 1; j < linecnt; j++ {
					jline := linebuf[j]
					lnaddr, err := lineaddr(jline)
					if pc == lnaddr {
						last_pc_line = jline
					}
				}
				found = true
				// retline = last_pc_line
				// dw.print_pcline(retline, pc)
				retfileno, retlineno = dw.pcfilelineno(line)
				break
			} else if prev_line != nil && pc > prev_lineaddr && pc < lnaddr {
				found = true
				// retline = prev_line
				// dw.print_pcline(retline, pc)
				retfileno, retlineno = dw.pcfilelineno(line)
				break
			}
			islne, err2 := lineendsequence(line)
			if islne == ctrue {
				prev_line = 0
			} else {
				prev_lineaddr = lnaddr
				prev_line = line
			}
		}
		break
	}
	srclines_dealloc2(linectx)

	return
}
func (dw *Dwarf) print_pcline(line Line, pc Addr) {
	var filename = "???"
	var fileline int
	if line != nil {
		retname, err1 := linesrc(line)
		if err1.Fail() {
			println(err1.Error())
		} else {
			filename = retname
		}
		retlineno, err2 := lineno(line)
		fileline = retlineno
	}
	println(pc, filename, fileline, line)
}

// from cached srcfiles, if line object gone
func (dw *Dwarf) print_pcline2(line Line, pc Addr, cuidx int) (string, int) {
	var filename = "???"
	var fileline int
	if line != nil {
		retlineno, err2 := lineno(line)
		fileline = retlineno

		retname, err1 := linesrc(line)
		if err1.Fail() {
			println(err1.Error())
		} else {
			filename = retname
		}

		// try by fileno
		fileno, err3 := line_srcfileno(line)
		if err3.Fail() {
			println(err3.Error())
		} else {
			fidxs := dw.a2l.cufilesm[cuidx]
			vidx := fidxs[fileno-1]
			retname := dw.a2l.cufilesv[vidx]
			filename = retname
		}
	}
	println(cuidx, pc, filename, fileline, line)
	return filename, fileline
}
func (dw *Dwarf) pcfilelineno(line Line) (fileno int, lineno1 int) {
	retlineno, err2 := lineno(line)
	lineno1 = retlineno
	retfileno, err3 := line_srcfileno(line)
	if err3.Fail() {
		println(err3.Error())
	} else {
		fileno = retfileno
	}
	return
}
func (dw *Dwarf) fileno2file(cuidx int, fileno int) string {
	a2l := dw.a2l
	fidxs := a2l.cufilesm[cuidx]
	vidx := fidxs[fileno-1]
	retname := a2l.cufilesv[vidx]
	return retname
}
func (dw *Dwarf) check_lookup_table() {
	a2l := dw.a2l
	tabsz := usize(a2l.maxpc - a2l.minpc)
	nilpcln := 0
	for i := 0; i < tabsz; i++ {
		line := a2l.pclntab[i]
		if line == nil {
			// println(tabsz, i, line)
			nilpcln++
		}
	}
	nilctx := 0
	for i := 0; i < a2l.ctxts.len(); i++ {
		ctx := a2l.ctxts[i]
		if ctx == nil {
			// println(a2l.cucnt, i, ctx)
			nilctx++
		}
	}
	println(a2l.cucnt, nilctx, tabsz, nilpcln)
}

func (dw *Dwarf) calc_lookup_table() {
	a2l := dw.a2l

	cuit := dw.NewCUIterm()
	for {
		cuhdr, err1 := cuit.Next()
		if err1.Fail() {
			break
		}
		cudie := cuhdr.CUdie
		cuidx := cuhdr.Index

		verout, tabcnt, linectx, err2 := srclines2(cudie)
		if err2.Okay() {
			a2l.ctxts[cuidx] = linectx
			if tabcnt == 1 { // what the 1?
				linebuf, linecnt, err := srclines_from_linecontext(linectx)
				if err.Fail() {
					println(err.Error())
					srclines_dealloc2(linectx)
				}

				var prev_lineaddr Addr
				var prev_line Line
				for i := 0; i < linecnt; i++ {
					line := linebuf[i]
					lineaddr, err := lineaddr(line)
					// println(cuit.idx, i, lineaddr)
					if prev_line != nil {
						for addr := prev_lineaddr; addr < lineaddr; addr++ {
							tabidx := addr - a2l.minpc
							a2l.pclntab[tabidx] = linebuf[i-1]
						}
						fillcnt := int(lineaddr - prev_lineaddr)
						// println(cuit.idx, i, fillcnt)
					}

					islnend, err2 := lineendsequence(line)
					if islnend == ctrue {
						prev_line = 0
					} else {
						prev_lineaddr = lineaddr
						prev_line = line
					}
				}
			}
		}

		dw.dealloc(cudie, DW_DLA_DIE)
	}
}

func (dw *Dwarf) calc_minmax_pc() {
	a2l := dw.a2l

	cuit := dw.NewCUIterm()
	for {
		cuhdr, err1 := cuit.Next()
		if err1.Fail() {
			break
		}
		cudie := cuhdr.CUdie

		minpc, err2 := lowpc(cudie)
		if err2.Okay() {
			a2l.setminpc(minpc)
		}
		maxpc, err3 := highpcx(cudie)
		if err3.Okay() {
			a2l.setmaxpc(maxpc)
		}

		//
		rgattr, err4 := attr(cudie, DW_AT_ranges)
		if err4.Okay() {
			offset, err5 := global_formref(rgattr)
			if err5.Okay() {
				var baseaddr Addr
				if a2l.minpc != DW_DLV_BADADDR {
					baseaddr = a2l.minpc
				}

				ranges, rgcnt, bytecnt, err := get_ranges1(dw.dbg, offset, cudie)
				for i := 0; i < rgcnt; i++ {
					// cur := ranges + i // TODO compiler
					var cur *Ranges
					cur = usize(ranges) + usize(i*sizeof(Ranges))
					if cur.dwr_type == DW_RANGES_ENTRY {
						rglowpc := baseaddr + cur.dwr_addr1
						rghighpc := baseaddr + cur.dwr_addr2
						a2l.setminpc(rglowpc)
						a2l.setmaxpc(rghighpc)
					} else if cur.dwr_type == DW_RANGES_ADDRESS_SELECTION {
						baseaddr = cur.dwr_addr2
					} else {
						baseaddr = a2l.minpc
					}
				}
				ranges_dealloc(dw.dbg, ranges, rgcnt)
			}
			dw.dealloc(rgattr, DW_DLA_ATTR)
		}
		dw.dealloc(cudie, DW_DLA_DIE)
	}
	var cucnt int = cuit.idx
	a2l.cucnt = cucnt
	println("minpc", a2l.minpc, "maxpc", a2l.maxpc, DW_DLV_BADADDR, cucnt) // why output 27-nilty???
}

func (dw *Dwarf) calc_cufiles() {
	a2l := dw.a2l

	// TODO compiler not support string key
	fileidxs := map[string]int{} // name => index
	cuit := dw.NewCUIterm()
	for {
		cuhdr, err1 := cuit.Next()
		if err1.Fail() {
			break
		}
		cudie := cuhdr.CUdie
		cuidx := cuhdr.Index

		files, err2 := srcfiles(cudie)
		fidxs := []int{}
		for idx, filename := range files {
			ok := fileidxs.haskey(filename)
			fidx := fileidxs[filename]
			if !ok {
				a2l.cufilesv = append(a2l.cufilesv, filename)
				fidx = a2l.cufilesv.len() - 1
				fileidxs[filename] = fidx
			}
			// a2l.cufilesm[cuidx] = append(a2l.cufilesm[cuidx], fidx)// TODO compiler
			fidxs = append(fidxs, fidx)
		}
		a2l.cufilesm[cuidx] = fidxs
		dw.dealloc(cudie, DW_DLA_DIE)
	}
	var cucnt int = cuit.idx
	a2l.cucnt = cucnt
	println("cufiles", cucnt, fileidxs.len()) // why output 27-nilty???
}
func (dw *Dwarf) check_cufiles() {
	a2l := dw.a2l
	filesvlen := a2l.cufilesv.len()
	filesmlen := a2l.cufilesm.len()
	for i := 0; i < filesvlen; i++ {
		file1 := a2l.cufilesv[i]
		// println(i, file1)
	}
	println("cufiles", "vcnt", filesvlen, "mcnt", filesmlen)
}

///
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

func (dwerr Error) Okay() bool { return dwerr == nil }
func (dwerr Error) Fail() bool { return dwerr != nil }
func (dwerr Error) Errno() int {
	if dwerr == ErrNoEntry {
		return DW_DLV_NO_ENTRY
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
