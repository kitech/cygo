package main

/*
尝试解析 C code 为ast

tcc 做 preprocessor, 然后 tree-sitter 解析为类似 ast的结构，再加部分自己写的代码，提取所需的节点信息

或者用 go-clang
*/

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"gopp"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	sitter "github.com/smacker/go-tree-sitter"
	trspc "github.com/smacker/go-tree-sitter/c"
)

type cparser1 struct {
	name string

	srcbuf    string
	ppsrcfile string
	ppsrc     []byte
	pplines   []string
	clrlines  []string

	prsit *sitter.Parser
	trn   *sitter.Tree

	hdrfiles map[string]int // filepath => lineno, reorder
}
type stfieldlist map[string]*stfield
type stfield struct {
	name  string
	tystr string
	tyobj types.Type
	idx   int
}

func newcparser1(name string) *cparser1 {
	cp := &cparser1{}
	cp.name = name
	cp.prsit = sitter.NewParser()
	cp.prsit.SetLanguage(trspc.GetLanguage())

	cp.hdrfiles = map[string]int{}
	return cp
}

const (
	csym_non = iota + 999
	csym_file
	csym_define
	csym_enum
	csym_var
	csym_func
	csym_struct
	csym_field
	csym_type
)

func csym_kind2str(kind int) string {
	kinds := map[int]string{
		csym_non:    "non",
		csym_file:   "file",
		csym_define: "define",
		csym_enum:   "enum",
		csym_var:    "var",
		csym_func:   "func",
		csym_struct: "struct",
		csym_field:  "field",
		csym_type:   "type",
	}
	if str, ok := kinds[kind]; ok {
		return str
	}
	return fmt.Sprintf("csymunk%d", kind)
}

type csymdata struct {
	kind   int
	name   string
	tyval  string
	tyobj  types.Type
	define ast.Expr
	struc  stfieldlist
}

func newcsymdata(name string, kind int) *csymdata {
	d := &csymdata{}
	d.name = name
	d.kind = kind
	return d
}

// 当前进程有效
type cparser1cache struct {
	// 当前进程生成的preprocessor文件表
	ppfiles  map[string]int // filepath => 1
	hdrfiles map[string]int // filepath => lineno
	csyms    map[string]*csymdata
}

func newcparser1cache() *cparser1cache {
	cpc := &cparser1cache{}
	cpc.ppfiles = map[string]int{}
	cpc.hdrfiles = map[string]int{}
	cpc.csyms = map[string]*csymdata{}
	return cpc
}

func (cp1c *cparser1cache) add(kind int, symname string, tyvalx interface{}) {
	symname = strings.Replace(symname, " ", "_", -1)
	if _, ok := cp1c.csyms[symname]; ok {
		return
	}
	csi := newcsymdata(symname, kind)
	switch kind {
	case csym_define:
		csi.define = tyvalx.(ast.Expr)
	case csym_struct:
		csi.struc = stfieldlist{}
	default:
		csi.tyval = tyvalx.(string)
	}
	cp1c.csyms[symname] = csi
}
func (cp1c *cparser1cache) add_field(stname string, fldname string, fldty string) *csymdata {
	stname = strings.Replace(stname, " ", "_", -1)
	csi := cp1c.csyms[stname]
	csi.struc[fldname] = &stfield{fldname, fldty, nil, len(csi.struc)}
	return csi
}
func (cp1c *cparser1cache) getsym(symname string) (*csymdata, bool) {
	symname = strings.Replace(symname, " ", "_", -1)
	csi, ok := cp1c.csyms[symname]
	return csi, ok
}

var cp1cache = newcparser1cache()

func rmoldtccppfiles() {
	files, err := filepath.Glob("/tmp/tcctrspp*.c")
	gopp.ErrPrint(err)
	for _, filename := range files {
		if _, ok := cp1cache.ppfiles[filename]; ok {
			continue
		}
		err := os.Remove(filename)
		gopp.ErrPrint(err, filename)
	}
}
func (cp *cparser1) parsestr(code string) error {
	rmoldtccppfiles()
	filename := fmt.Sprintf("/tmp/tcctrspp.%s.%d.c", cp.name, rand.Intn(10000000)+50000)
	cp1cache.ppfiles[filename] = 1

	// TODO hard code path
	incdirs := preincdirs
	code = codepfx + code
	btime := time.Now()
	err := tccpp(code, filename, incdirs)
	gopp.ErrPrint(err, cp.name, filename)
	log.Println("tccpp", cp.name, err, time.Since(btime))
	if err != nil {
		return err
	}

	bcc, err := ioutil.ReadFile(filename)
	gopp.ErrPrint(err, filename)
	cp.ppsrc = bcc
	// defer os.Remove(filename)

	// clean code to make tree sitter happy
	btime = time.Now()
	cp.pplines = strings.Split(string(cp.ppsrc), "\n")
	cp.cltfiles()
	bcc = []byte(strings.Join(cp.clrlines, "\n"))
	cp.ppsrc = bcc
	cp.pplines = cp.clrlines
	cp.clrlines = nil
	log.Println("clrpp", cp.name, time.Since(btime))

	btime = time.Now()
	trn := cp.prsit.Parse(bcc)
	cp.trn = trn
	log.Println("trsit parse", cp.name, time.Since(btime))

	cp.collect()
	return err
}

func (cp *cparser1) parsefile(filename string) bool {
	return true
}

func (cp *cparser1) fill_hotfixs() {
	cp1cache.add(csym_enum, "__LINE__", "int")
	cp1cache.add(csym_var, "__FILE__", "char*")
	cp1cache.add(csym_var, "__FUNCTION__", "char*")
	cp1cache.add(csym_var, "__func__", "char*")
	cp1cache.add(csym_var, "errno", "int")
	cp1cache.add(csym_var, "NULL", "void*")
}
func (cp *cparser1) collect() {
	// cp.pplines = strings.Split(string(cp.ppsrc), "\n")
	// cp.cltfiles()
	cp.fill_hotfixs()
	cp.cltdefines()
	btime := time.Now()
	cp.walk(cp.trn.RootNode(), 0) // TODO slow
	results := map[string]interface{}{
		"hdrfiles": len(cp.hdrfiles),
		"csyms":    len(cp1cache.csyms),
	}
	log.Println(cp.name, results, len(cp1cache.csyms), time.Since(btime))
}

func (cp *cparser1) walk(n *sitter.Node, lvl int) {
	var txt string

	switch n.Type() {
	case "declaration":
		fallthrough
	case "function_definition":
		fallthrough
	case "enumerator":
		fallthrough
	case "struct_specifier":
		fallthrough
	case "field_declaration":
		fallthrough
	case "type_definition": // typedef xxx yyy;
		fallthrough
	case "primitive_type":
		fallthrough
	case "assignment_expression":
		txt = cp.exprtxt(n)
		txt = strings.TrimSpace(txt)
	}

	switch n.Type() {
	case "declaration":
		gopp.Assert(len(txt) > 0, "wtfff", txt)

		// log.Println(n.Type(), n.ChildCount(), len(txt), txt)
		isfunc := false
		nc := int(n.ChildCount())
		for i := 0; i < nc; i++ {
			nx := n.Child(i)
			// log.Println(n.Type(), i, n.Child(i).Type(), len(txt), txt)
			if nx.Type() == "function_declarator" {
				isfunc = true
				break
			}
		}
		if strings.HasSuffix(txt, ";") {
			txt2 := strings.TrimRight(txt, ";")
			txt2 = strings.TrimSpace(txt2)
			if strings.HasSuffix(txt2, ")") {
				isfunc = true
			}
		}
		declkind := gopp.IfElseStr(isfunc, "func", "var")
		// func
		if isfunc {
			funcname, functype := getfuncname(txt)
			// log.Println(n.Type(), declkind, functype, "//", funcname, txt+"//")
			cp1cache.add(csym_func, funcname, functype)
		} else {
			// var
			funcname, functype := getvarname(txt)
			// log.Println(n.Type(), declkind, functype, "//", funcname, txt+"//", )
			cp1cache.add(csym_var, funcname, functype)
		}

		if false {
			log.Println(n.Type(), n.ChildCount(), declkind, len(txt), txt)
		}
	case "function_definition":
		ipos := strings.Index(txt, "{")
		declstr := strings.TrimSpace(txt[:ipos])
		funcname, functype := getfuncname(declstr + ";")
		cp1cache.add(csym_func, funcname, functype)
	case "enumerator":
		if false {
			log.Println(n.Type(), len(txt), txt)
		}
		ipos := strings.Index(txt, " ")
		var fld0, fld1 string
		if ipos < 0 {
			fld0 = txt
		} else {
			fld0 = txt[0:ipos]
			fld1 = txt[ipos+1:]
		}
		cp1cache.add(csym_enum, fld0, fld1)
	case "struct_specifier":
		gopp.Assert(len(txt) > 0, "wtfff", txt)
		// if _, ok := cp.structs[txt]; ok {
		// 	break
		// }
		if _, ok := cp1cache.csyms[txt]; ok {
			break
		}

		stname := strings.Split(txt, "{")[0]
		stname = strings.TrimSpace(stname)
		cp1cache.add(csym_struct, stname, stfieldlist{})
		if false {
			log.Println(n.Type(), len(txt), txt)
		}
	case "field_declaration":
		gopp.Assert(len(txt) > 0, "wtfff", txt)

		pn := n.Parent()   // field_declaration_list
		ppn := pn.Parent() // struct_specifier
		if ppn.Type() == "translation_unit" ||
			ppn.Type() == "union_specifier" {
			break
		}
		gopp.Assert(ppn.Type() == "struct_specifier", "wtfff", ppn.Type())

		stbody := cp.exprtxt(ppn)
		stname := strings.Split(stbody, "{")[0]
		stname = strings.TrimSpace(stname)
		_, instruct := cp1cache.getsym(stname)
		gopp.Assert(instruct, "wtfff", stname)

		fldname, tystr := getvarname(txt)
		if strings.Contains(txt, "[") {
			// char foo[bar] format
			str1 := strings.Split(txt, "[")[0]
			arr1 := strings.Split(str1, " ")
			fldname = arr1[len(arr1)-1]
			tystr = strings.Join(arr1[:len(arr1)-1], " ")
			tystr += strings.Repeat("*", strings.Count(txt, "["))
		}
		csi := cp1cache.add_field(stname, fldname, tystr)
		fldcnt := len(csi.struc)
		if false {
			log.Println(n.Type(), len(txt), txt, ppn.Type(),
				instruct, stname, fldcnt, "//", tystr, "//", fldname)
		}
	case "type_definition": // typedef xxx yyy;
		gopp.Assert(len(txt) > 0, "wtfff", txt)

		txt = strings.TrimRight(txt, ";")
		txt = strings.TrimSpace(txt)
		// func type
		isfuncty := strings.Contains(txt, " (*") &&
			!strings.Contains(txt, "{")
		if isfuncty {
			reg := regexp.MustCompile(`.* \(\*(.+)\).*`)
			mats := reg.FindAllStringSubmatch(txt, -1)
			tyname := mats[0][1]
			cp1cache.add(csym_type, tyname, "void*")
		} else {
			fields := strings.Split(txt, " ")
			tyname := fields[len(fields)-1]
			realty := strings.Join(fields[1:len(fields)-1], " ")
			// log.Println(n.Type(), len(fields), fields, tyname, realty)
			cp1cache.add(csym_type, tyname, realty)
		}
		if false {
			log.Println(n.Type(), len(txt), txt)
		}
	case "assignment_expression":
		pn := n.Parent()
		ppn := pn.Parent()
		pppn := ppn.Parent()
		fields := strings.Split(txt, "=")
		ve, err := parser.ParseExpr(fields[1])
		gopp.ErrPrint(err, txt)
		if err == nil {
			idtname := strings.TrimSpace(fields[0])
			cp1cache.add(csym_define, idtname, ve)
		}
		if false {
			log.Println(n.Type(), pn.Type(), ppn.Type(), pppn.Type(), len(txt), txt)
		}
	case "primitive_type":
		cp1cache.add(csym_type, txt, txt)
	case "translation_unit": // full text
	default:
		if false {
			txt := cp.exprtxt(n)
			log.Println(n.Type(), len(txt), txt)
		}
	}
	if false {
		txt := cp.exprtxt(n)
		log.Println(n.Type(), len(txt), txt)
	}

	brn := int(n.ChildCount())
	for i := 0; i < brn; i++ {
		nx := n.Child(i)
		cp.walk(nx, lvl+1)
	}
}

func (cp *cparser1) exprtxt(n *sitter.Node) string {
	bpos := n.StartPoint()
	epos := n.EndPoint()

	txt := ""
	for i := bpos.Row; ; i++ {
		isfirst := i == bpos.Row
		islast := i == epos.Row
		line := cp.pplines[i]
		bcol := int(bpos.Column)
		ecol := int(epos.Column)
		if isfirst && islast {
			txt = line[bcol:ecol]
		} else if isfirst {
			txt = line[bcol:]
		} else if islast {
			txt += line[:ecol]
		} else {
			if strings.HasPrefix(line, "#") {
			} else {
				txt += line
			}
		}
		if i >= epos.Row {
			break
		}
	}
	return txt
}

func (cp *cparser1) cltfiles() {
	btime := time.Now()
	clrlines := []string{} // without # line and make tree sitter happy
	for idx, line := range cp.pplines {
		if strings.HasPrefix(line, "# ") {
			// log.Println("header file?", idx, line)
			fields := strings.Split(line, " ")
			hdrfile := strings.Trim(fields[2], "\"<>")
			if _, ok := cp.hdrfiles[hdrfile]; !ok {
				cp.hdrfiles[hdrfile] = idx
			}
		} else {
			clrlines = append(clrlines, line)
		}
	}
	cp.clrlines = clrlines
	log.Println("files", len(cp.hdrfiles), "left", len(clrlines), time.Since(btime))
}

func (cp *cparser1) cltdefines() {
	btime := time.Now()
	for hdrfile, _ := range cp.hdrfiles {
		if _, ok := cp1cache.ppfiles[hdrfile]; ok {
			continue
		}
		cp1cache.ppfiles[hdrfile] = 1

		bcc, err := ioutil.ReadFile(hdrfile)
		gopp.ErrPrint(err, hdrfile)
		lines := strings.Split(string(bcc), "\n")
		for _, line := range lines {
			if !strings.HasPrefix(line, "#define ") &&
				!strings.HasPrefix(line, "# define ") {
				continue
			}
			line = trimcomment1(line)
			line = refmtdefineline(line)
			fields := strings.Split(line, " ")
			// log.Println("define?", line, len(fields), fields, strings.Contains(line, "\t"))
			if len(fields) == 2 {
				// bool
				ve, err := parser.ParseExpr("true")
				gopp.Assert(err == nil, "wtfff", line)
				cp1cache.add(csym_define, fields[1], ve)
				continue
			}

			defname := fields[1]
			defval := strings.Join(fields[2:], " ")
			// log.Println("define?", defname, "=", defval, line, fields)
			codeline := strings.TrimSpace(defval)
			ve, err := parser.ParseExpr(codeline)
			if false {
				gopp.ErrPrint(err, ve, "/", codeline, "/", line, len(fields))
			}
			if err == nil {
				// log.Println(ve, reftyof(ve), codeline)
				cp1cache.add(csym_define, defname, ve)
			} else {
				// log.Println(ve, reftyof(ve), defname, codeline, line)
				if !strings.Contains(defval, " ") {
					pardef, ok := cp1cache.getsym(defname)
					log.Println(defname, pardef, ok)
					// cp.defines[defname] = defval
				}
			}
		}
	}
	log.Println("defines", time.Since(btime))
}

func trimcomment1(line string) string {
	bpos := strings.Index(line, "/*")
	epos := strings.Index(line, "*/")
	if bpos >= 0 && epos > bpos {
		return line[0:bpos] + line[epos+2:]
	}
	return line
}
func refmtdefineline(line string) string {
	str := ""
	lastsp := false
	lastsharp := false
	var lastch rune
	for _, ch := range line {
		if ch == ' ' {
			if lastsp {
			} else if lastsharp {
			} else {
				str += string(ch)
			}
		} else if ch == '\t' {
			if !lastsp {
				str += " "
			}
			lastsp = true
		} else {
			str += string(ch)
			lastsp = false
		}
		if ch == '#' {
			lastsharp = true
		} else {
			lastsharp = false
		}
		lastch = ch
	}
	if lastch == ' ' {
		str = strings.TrimSpace(str)
	}
	return str
}

// s : type funcname();
func getfuncname(s string) (string, string) {
	fields := strings.Split(s, "(")
	gopp.Assert(len(fields) > 1, "wtfff", s)
	fields2 := strings.Split(strings.TrimSpace(fields[0]), " ")
	fields3 := []string{}
	for _, fld := range fields2 {
		if fld == "extern" || fld == "const" {
			continue
		}
		fields3 = append(fields3, fld)
	}
	fields2 = fields3
	//log.Println(s, len(fields2), fields2)

	tyname := strings.Join(fields2[0:len(fields2)-1], " ")
	funcname := fields2[len(fields2)-1]
	for {
		if strings.HasPrefix(funcname, "*") {
			tyname += "*"
			funcname = funcname[1:]
		} else {
			break
		}
	}
	return funcname, tyname
}

// s : type funcname();
func getvarname(s string) (string, string) {
	fields := strings.Split(s, ";")
	gopp.Assert(len(fields) > 1, "wtfff", s)
	fields2 := strings.Split(strings.TrimSpace(fields[0]), " ")
	fields3 := []string{}
	for _, fld := range fields2 {
		if fld == "extern" || fld == "const" {
			continue
		}
		fields3 = append(fields3, fld)
	}
	fields2 = fields3
	//log.Println(s, len(fields2), fields2)

	tyname := strings.Join(fields2[0:len(fields2)-1], " ")
	funcname := fields2[len(fields2)-1]
	for {
		if strings.HasPrefix(funcname, "*") {
			tyname += "*"
			funcname = funcname[1:]
		} else {
			break
		}
	}
	return funcname, tyname
}

func (cp *cparser1) symtype(sym string) (tystr string, tyobj types.Type) {
	csi, incache := cp1cache.getsym(sym)
	log.Println(cp.name, sym, "incache", incache)
	var symkind ast.ObjKind = ast.Bad
	_ = symkind

	if incache {
		switch csi.kind {
		case csym_define, csym_enum:
			symkind = ast.Con
		case csym_var:
			symkind = ast.Var
		case csym_func:
			symkind = ast.Fun
		case csym_type:
			symkind = ast.Typ
		}
		log.Println(cp.name, sym, "incache", incache, symkind)

		switch csi.kind {
		case csym_define:
			defexpr := csi.define
			switch ety := defexpr.(type) {
			case *ast.BasicLit:
				switch ety.Kind {
				case token.INT:
					tyobj = types.Typ[types.UntypedInt]
				default:
					log.Println("todo", cp.name, sym, defexpr, reftyof(ety), ety.Kind)
				}
			case *ast.BinaryExpr:
				switch xe := ety.X.(type) {
				case *ast.BasicLit:
					switch xe.Kind {
					case token.INT:
						tyobj = types.Typ[types.UntypedInt]
					default:
						log.Println("todo", cp.name, sym, defexpr, reftyof(ety), xe.Kind)
					}
				default:
					log.Println("todo", cp.name, sym, defexpr, reftyof(ety), reftyof(xe))
				}
			case *ast.Ident:
				if sym == ety.Name {
					_, tyobj = cp.ctype2go2(sym, csi)
					break
				}
				log.Println("redir", sym, "=>", ety.Name)
				return cp.symtype(ety.Name)
			default:
				vev := types.ExprString(defexpr)
				log.Println("todo", cp.name, sym, defexpr, reftyof(ety), vev)
			}
			log.Println(cp.name, sym, "incache", incache, symkind, tyobj)
		case csym_enum:
			tyobj = types.Typ[types.UntypedInt]
			return
		case csym_var, csym_func, csym_type:
			tystr, tyobj = cp.ctype2go2(sym, csi)
			return
		case csym_struct:
			log.Println(sym, csi.tyval)
			if csi.tyval == "" { // why
				csi.tyval = sym
			}
			tyobj = cp.tostructy(csi) // sym
			return
		default:
			log.Println(cp.name, sym, "incache", incache, csi.kind)
		}

	}

	return
}

// foo* => foo
func (cp *cparser1) topointee(csi *csymdata) *csymdata {
	tystr := csi.tyval
	gopp.Assert(strings.HasSuffix(tystr, "*"), "wtfff", tystr)

	newcsi := *csi
	newcsi.tyval = tystr[:len(tystr)-1]
	return &newcsi
}
func (cp *cparser1) ctype2go2(sym string, csi *csymdata) (
	tystr2 string, tyobj types.Type) {
	tystr2, tyobj = cp.ctype2go(sym, csi.tyval)
	if tyobj != nil {
		log.Println("non primitive_type?", sym, tystr2)
		return
	}

	log.Println("non primitive_type?", sym, "//", csi.tyval, "//", tystr2)
	tystr := csi.tyval
	if tystr == "" && sym != "" {
		tystr = sym
	}
	if strings.HasPrefix(tystr, "struct ") {
		log.Println("non primitive_type?", sym, tystr2)
		if strings.HasSuffix(tystr, "*") {
			// tyobj = types.Typ[types.Voidptr]
			newcsi := cp.topointee(csi)
			tystr3, tyobj3 := cp.ctype2go2(sym, newcsi)
			tystr2 = tystr3
			tyobj = types.NewPointer(tyobj3)
			return
		} else {
			// POD
			log.Println("non primitive_type?", sym, tystr2)
			tyobj = cp.tostructy(csi)
			return
		}
	}
	log.Println("non primitive_type?", sym, tystr2)

	return
}
func (cp *cparser1) ctype2go(sym, tystr string) (tystr2 string, tyobj types.Type) {
	log.Println(cp.name, sym, tystr)
	tystr = strings.TrimSpace(tystr)
	tystr2 = tystr

	switch tystr {
	case "cxstring*":
		tyobj = types.Typ[types.String]
	case "char*", "char *":
		tyobj = types.Typ[types.Byteptr]
	case "unsigned char*":
		tyobj = types.Typ[types.Byteptr]
	case "char**":
		tyobj = types.NewPointer(types.Typ[types.Byteptr])
	case "void *", "void*":
		tyobj = types.Typ[types.Voidptr]
	case "void":
		tyobj = (*types.Tuple)(nil)
	case "int*", "int *":
		tyobj = types.NewPointer(types.Typ[types.Int])
	case "long long int":
		tyobj = types.Typ[types.Int64]
	case "long int", "long":
		tyobj = types.Typ[types.Int64]
	case "unsigned long", "unsigned long int", "ulong":
		tyobj = types.Typ[types.Uint64]
	case "unsigned long long", "unsigned long long int", "ulonglong":
		tyobj = types.Typ[types.Uint64]
	case "unsigned", "unsigned int", "uint":
		tyobj = types.Typ[types.Uint]
	case "int":
		tyobj = types.Typ[types.Int]
	case "char":
		tyobj = types.Typ[types.Byte]
	case "uint16_t", "unsigned short", "unsigned short int", "ushort":
		tyobj = types.Typ[types.Uint16]
	case "int16_t", "short", "short int":
		tyobj = types.Typ[types.Int16]
	case "size_t", "time_t", "uintptr_t":
		tyobj = types.Typ[types.Usize]
	case "double":
		tyobj = types.Typ[types.Float64]
	case "float":
		tyobj = types.Typ[types.Float32]
	case "_Bool":
		tyobj = types.Typ[types.Bool]

	default:
		if sym == "SOCK_STREAM" {
			tyobj = types.Typ[types.Int] // TODO auto detect
			return
		}
		if strings.HasSuffix(tystr, "*") {
			starcnt := strings.Count(tystr, "*")
			canty := strings.TrimRight(tystr, "*")
			csi, ok := cp1cache.csyms[canty]
			if ok {
				undty := csi.tyval
				newty := undty + strings.Repeat("*", starcnt)
				log.Println(sym, tystr, "=>", newty)
				return cp.ctype2go(sym, newty)
			}
		}
		log.Println(cp.name, sym, tystr)
		csi, ok := cp1cache.csyms[tystr]
		if ok && strings.HasPrefix(csi.tyval, "enum {") {
			tyobj = types.Typ[types.Int]
			return
		} else if ok && csi.tyval != tystr {
			log.Println(cp.name, sym, tystr)
			return cp.ctype2go(sym, csi.tyval)
		}
		log.Println(cp.name, sym, tystr, cp1cache.csyms[tystr], cp1cache.csyms[sym])
	}
	return
}

func (cp *cparser1) getstruct(sym string) *csymdata {
	csi, incache := cp1cache.getsym(sym)
	log.Println(cp.name, sym, incache, csi != nil)
	cp.fillstructy(sym, csi)
	return csi
}
func (cp *cparser1) fillstructy(sym string, csi *csymdata) {
	if csi == nil {
		return
	}
	for fldname, fldo := range csi.struc {
		fldty, tyobj := cp.ctype2go(fldname, fldo.tystr)
		log.Println(cp.name, sym, fldname, fldo.tystr, "//", fldty, "//", tyobj, tyobj != nil)
		if tyobj != nil {
			fldo.tyobj = tyobj
		} else {
			log.Println(sym, fldo.name, fldo.tystr)
			if strings.HasPrefix(fldo.tystr, "struct ") {
				csi2, ok := cp1cache.getsym(fldo.tystr)
				if ok {
					cp.fillstructy(fldname, csi2)
					log.Println(sym, fldo.name, fldo.tystr, csi2.tyobj)
				} else {
					log.Println(sym, fldo.name, fldo.tystr)
				}
			} else {
				log.Println(sym, fldo.name, fldo.tystr)
			}
		}
	}
	return
}

func (cp *cparser1) tostructy(csi *csymdata) types.Type {
	tystr := csi.tyval
	if tystr == "" {
		tystr = csi.name
	}
	tystr2 := strings.ReplaceAll(tystr, "struct ", "struct_")
	gopp.Assert(strings.HasPrefix(tystr2, "struct_"), "wtfff", tystr, tystr2)
	stname := tystr2

	var fldvars []*types.Var
	for fldname, fldo := range csi.struc {
		log.Println(stname, fldname, fldo.tystr, fldo.tyobj)
		var tyobj types.Type
		_, tyobj = cp.ctype2go(fldo.tystr, fldo.tystr)
		csi2, incache := cp1cache.getsym(fldo.tystr)
		if incache && tyobj == nil {
			_, tyobj = cp.ctype2go2(fldo.tystr, csi2)
		}
		if tyobj == nil {
			log.Println(stname, fldname, fldo.tystr, fldo.tyobj, incache)
		}

		fldvar := types.NewVar(token.NoPos, fcpkg, fldo.name, tyobj)
		fldvars = append(fldvars, fldvar)
	}
	sty1 := &types.Struct{}
	sty1 = types.NewStruct(fldvars, nil)
	// keep NewTypeName's type arg nil, so next step get a valid struct type
	stobj := types.NewTypeName(token.NoPos, fcpkg, stname, nil)
	stobj2 := types.NewNamed(stobj, sty1, nil)
	return stobj2
}
