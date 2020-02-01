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
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	trspc "github.com/smacker/go-tree-sitter/c"
)

type cparser1 struct {
	name string

	srcbuf    string
	ppsrcfile string
	ppsrc     []byte
	pplines   []string

	prsit *sitter.Parser
	trn   *sitter.Tree

	files   map[string]int         // filepath => lineno, reorder
	defines map[string]ast.Expr    // name => value
	enums   map[string]string      // name => value
	vars    map[string]string      // name => type
	funcs   map[string]string      // name => return type
	structs map[string]stfieldlist // name => fields
	types   map[string]string      // name => primitive type
}
type stfieldlist map[string]stfield
type stfield struct {
	name  string
	tystr string
}

func newcparser1(name string) *cparser1 {
	cp := &cparser1{}
	cp.name = name
	cp.prsit = sitter.NewParser()
	cp.prsit.SetLanguage(trspc.GetLanguage())

	cp.files = map[string]int{}
	cp.defines = map[string]ast.Expr{}
	cp.enums = map[string]string{}
	cp.vars = map[string]string{}
	cp.funcs = map[string]string{}
	cp.structs = map[string]stfieldlist{}
	cp.types = map[string]string{}
	return cp
}

// 当前进程生成的preprocessor文件表
var curprocppfiles = map[string]int{}

func rmoldtccppfiles() {
	files, err := filepath.Glob("/tmp/tcctrspp*.c")
	gopp.ErrPrint(err)
	for _, filename := range files {
		if _, ok := curprocppfiles[filename]; ok {
			continue
		}
		err := os.Remove(filename)
		gopp.ErrPrint(err, filename)
	}
}
func (cp *cparser1) parsestr(code string) bool {
	rmoldtccppfiles()
	filename := fmt.Sprintf("/tmp/tcctrspp.%d.c", rand.Intn(10000000)+50000)
	curprocppfiles[filename] = 1

	err := tccpp(code, filename, nil)
	gopp.ErrPrint(err, filename)

	bcc, err := ioutil.ReadFile(filename)
	gopp.ErrPrint(err, filename)
	cp.ppsrc = bcc
	// defer os.Remove(filename)

	trn := cp.prsit.Parse(bcc)
	cp.trn = trn

	cp.collect()
	return true
}

func (cp *cparser1) parsefile(filename string) bool {
	return true
}

func (cp *cparser1) fill_hotfixs() {
	cp.enums["__LINE__"] = "int"
	cp.vars["__FILE__"] = "char*"
	cp.vars["errno"] = "int"
}
func (cp *cparser1) collect() {
	cp.pplines = strings.Split(string(cp.ppsrc), "\n")
	cp.fill_hotfixs()
	cp.cltfiles()
	cp.cltdefines()
	cp.walk(cp.trn.RootNode(), 0)
	results := map[string]interface{}{
		"files":   len(cp.files),
		"defines": len(cp.defines),
		"enums":   len(cp.enums),
		"vars":    len(cp.vars),
		"funcs":   len(cp.funcs),
		"structs": len(cp.structs),
		"types":   len(cp.types),
	}
	log.Println(cp.name, results)
}

func (cp *cparser1) walk(n *sitter.Node, lvl int) {
	txt := cp.exprtxt(n)
	txt = strings.TrimSpace(txt)

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
			cp.funcs[funcname] = functype
		} else {
			// var
			funcname, functype := getvarname(txt)
			// log.Println(n.Type(), declkind, functype, "//", funcname, txt+"//", )
			cp.vars[funcname] = functype
		}

		if false {
			log.Println(n.Type(), n.ChildCount(), declkind, len(txt), txt)
		}

	case "enumerator":
		txt := cp.exprtxt(n)
		if false {
			log.Println(n.Type(), len(txt), txt)
		}
		fields := strings.Split(txt, " ")
		cp.enums[fields[0]] = strings.Join(fields[1:], " ")
	case "struct_specifier":
		gopp.Assert(len(txt) > 0, "wtfff", txt)
		if _, ok := cp.structs[txt]; ok {
			break
		}

		stname := strings.Split(txt, "{")[0]
		stname = strings.TrimSpace(stname)
		cp.structs[stname] = stfieldlist{}
		log.Println(n.Type(), len(txt), txt)
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
		_, instruct := cp.structs[stname]
		gopp.Assert(instruct, "wtfff", stname)

		fldname, tystr := getvarname(txt)
		cp.structs[stname][fldname] = stfield{fldname, tystr}
		fldcnt := len(cp.structs[stname])
		if true {
			log.Println(n.Type(), len(txt), txt, ppn.Type(), instruct, stname, fldcnt, "//")
		}
	case "type_definition": // typedef xxx yyy;
		gopp.Assert(len(txt) > 0, "wtfff", txt)

		txt = strings.TrimRight(txt, ";")
		txt = strings.TrimSpace(txt)
		// func type
		isfuncty := strings.Contains(txt, " (*")
		if isfuncty {
			// TODO
		} else {
			fields := strings.Split(txt, " ")
			cp.types[fields[len(fields)-1]] = strings.Join(fields[1:len(fields)-1], " ")
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
			cp.defines[idtname] = ve
		}
		if false {
			log.Println(n.Type(), pn.Type(), ppn.Type(), pppn.Type(), len(txt), txt)
		}
	default:
		txt := cp.exprtxt(n)
		if false {
			log.Println(n.Type(), len(txt), txt)
		}
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
	for idx, line := range cp.pplines {
		if strings.HasPrefix(line, "# ") {
			// log.Println("header file?", idx, line)
			fields := strings.Split(line, " ")
			hdrfile := strings.Trim(fields[2], "\"<>")
			if _, ok := cp.files[hdrfile]; !ok {
				cp.files[hdrfile] = idx
			}
		}
	}
	log.Println("files", len(cp.files))
}

func (cp *cparser1) cltdefines() {
	for hdrfile, _ := range cp.files {
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
				cp.defines[fields[1]] = ve
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
				cp.defines[defname] = ve
			} else {
				// log.Println(ve, reftyof(ve), defname, codeline, line)
				if !strings.Contains(defval, " ") {
					pardef, ok := cp.defines[defval]
					log.Println(defname, pardef, ok)
					// cp.defines[defname] = defval
				}
			}
		}
	}
	log.Println("defines", len(cp.defines))
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
	for _, ch := range line {
		t := string(ch)
		if t == " " {
			if lastsp {
			} else if lastsharp {
			} else {
				str += t
			}
		} else if t == "\t" {
			if !lastsp {
				str += " "
			}
			lastsp = true
		} else {
			str += t
			lastsp = false
		}
		if t == "#" {
			lastsharp = true
		} else {
			lastsharp = false
		}
	}
	str = strings.TrimSpace(str)
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
	defexpr, indefine := cp.defines[sym]
	_, inenum := cp.enums[sym]
	tystr1, infunc := cp.funcs[sym]
	tystr2, intype := cp.types[sym]
	tystr3, invar := cp.vars[sym]
	log.Println(cp.name, sym, "indefine", indefine, "inenums", inenum,
		"infuncs", infunc, "intypes", intype, "invars", invar)

	if indefine {
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
				break
			}
			log.Println("redir", sym, "=>", ety.Name)
			return cp.symtype(ety.Name)
		default:
			vev := types.ExprString(defexpr)
			log.Println("todo", cp.name, sym, defexpr, reftyof(ety), vev)
		}
	}

	if inenum {
		tyobj = types.Typ[types.UntypedInt]
		return
	}

	if infunc {
		tystr, tyobj = ctype2go(sym, tystr1)
		return
	}

	if intype {
		tystr, tyobj = ctype2go(sym, tystr2)
		return
	}

	if invar {
		tystr, tyobj = ctype2go(sym, tystr3)
		return
	}

	return
}

func ctype2go(sym, tystr string) (tystr2 string, tyobj types.Type) {
	log.Println(sym, tystr)
	tystr2 = tystr
	switch tystr {
	case "char*":
		tyobj = types.Typ[types.Byteptr]
	case "char**":
		tyobj = types.NewPointer(types.Typ[types.Byteptr])
	case "long int":
		tyobj = types.Typ[types.Int64]
	case "int":
		tyobj = types.Typ[types.Int]
	}
	return
}
