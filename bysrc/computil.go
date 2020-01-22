package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/printer"
	"go/token"
	"go/types"
	"log"
	"reflect"
	"strings"
	"unicode"

	"gopp"

	"golang.org/x/tools/go/ast/astutil"
)

// compile line context
type compContext struct {
	le  ast.Expr
	re  ast.Expr
	lty types.Type
	rty types.Type
}

func ismapty(tystr string) bool    { return strings.HasPrefix(tystr, "map[") }
func ismapty2(typ types.Type) bool { return ismapty(typ.String()) }
func isstrty(tystr string) bool {
	if strings.HasPrefix(tystr, "untyped ") {
		tystr = tystr[8:]
	}
	return tystr == "string"
}
func isstrty2(typ types.Type) bool {
	if typ == nil {
		log.Println("todo", typ)
		return false
	}
	return isstrty(typ.String())
}
func iscstrty2(typ types.Type) bool {
	tystr := typ.String()
	return strings.HasPrefix(tystr, "*") && strings.HasSuffix(tystr, "_Ctype_char")
}
func isslicety(tystr string) bool    { return strings.HasPrefix(tystr, "[]") }
func isslicety2(typ types.Type) bool { return isslicety(typ.String()) }
func isarrayty(tystr string) bool {
	s := ""
	for _, c := range tystr {
		if !unicode.IsDigit(c) {
			s += string(c)
		}
	}
	return strings.HasPrefix(s, "[]") && !strings.HasPrefix(tystr, "[]")
}
func isarrayty2(typ types.Type) bool   { return isarrayty(typ.String()) }
func iseface(tystr string) bool        { return strings.HasPrefix(tystr, "interface{}") }
func iseface2(typ types.Type) bool     { return typ != nil && iseface(typ.String()) }
func isiface(tystr string) bool        { return strings.HasPrefix(tystr, "interface{") }
func isiface2(typ types.Type) bool     { return typ != nil && isiface(typ.String()) }
func istypety(tystr string) bool       { return strings.HasPrefix(tystr, "type ") }
func istypety2(typ types.Type) bool    { return istypety(typ.String()) }
func ischanty(tystr string) bool       { return strings.HasPrefix(tystr, "chan ") }
func ischanty2(typ types.Type) bool    { return ischanty(typ.String()) }
func isvarty(tystr string) bool        { return strings.HasPrefix(tystr, "var ") }
func isvarty2(typ types.Type) bool     { return isvarty(typ.String()) }
func isstructty(tystr string) bool     { return strings.Contains(tystr, "/.") } // struct ???
func isstructty2(typ types.Type) bool  { return isstructty(typ.String()) }
func isinvalidty(tystr string) bool    { return strings.HasPrefix(tystr, "invalid ") }
func isinvalidty2(typ types.Type) bool { return isinvalidty(typ.String()) }
func isuntypedty(tystr string) bool    { return strings.HasPrefix(tystr, "untyped ") }
func isuntypedty2(typ types.Type) bool { return isuntypedty(typ.String()) }
func iswrapcfunc(name string) bool     { return strings.HasPrefix(name, "_Cfunc") }
func istuple(tystr string) bool        { return strings.Contains(tystr, "_multiret_") }
func iscident(e ast.Expr) bool {
	if idt, ok := e.(*ast.Ident); ok {
		return idt.Name == "C"
	}
	return false
}
func isctydeftype2(typ types.Type) bool {
	return strings.HasSuffix(typ.String(), "__ctype")
}
func iscsel222(e ast.Expr) bool {
	if se, ok := e.(*ast.SelectorExpr); ok {
		if iscident(se.X) {
			return true
		}
	}
	return false
}
func isnilident(e ast.Expr) bool {
	if idt, ok := e.(*ast.Ident); ok {
		return idt.Name == "nil"
	}
	return false
}

func iserrorty2(typ types.Type) bool {
	if typ == nil {
		return false
	}
	segs := strings.Split(typ.String(), ".")
	if len(segs) == 1 {
		return segs[0] == "error"
	}
	return segs[len(segs)-1] == "error"
}

func newLitInt(v int) *ast.BasicLit {
	return &ast.BasicLit{Kind: token.INT, Value: fmt.Sprintf("%d", v)}
}
func newLitStr(v string) *ast.BasicLit {
	return &ast.BasicLit{Kind: token.STRING, Value: fmt.Sprintf("%s", v)}
}
func newLitFloat(v float32) *ast.BasicLit {
	return &ast.BasicLit{Kind: token.FLOAT, Value: fmt.Sprintf("%f", v)}
}
func newIdent(v string) *ast.Ident {
	idt := &ast.Ident{}
	idt.Name = v
	idt.NamePos = token.NoPos
	return idt
}

func typesty2str(typ types.Type) string {
	ret := ""
	switch aty := typ.(type) {
	case *types.Basic:
		ret = fmt.Sprintf("%v", typ)
		ret = strings.Replace(ret, ".", "_", 1) // unsafe.Pointer
	case *types.Interface:
		return gopp.IfElseStr(aty.NumMethods() > 0, "cxiface", "cxeface")
	default:
		gopp.G_USED(aty)
		log.Println("todo", typ, reflect.TypeOf(typ))
		ret = fmt.Sprintf("%v", typ)
	}
	return ret
}

func trimCtype(s string) string {
	if strings.HasPrefix(s, "_Ctype_") {
		return s[7:]
	}
	return s
}

// used only when cannot found a valid types.Type
func iscsel(e ast.Expr) bool {
	// log.Println(e, reflect.TypeOf(e))
	switch te := e.(type) {
	case *ast.StarExpr:
		return iscsel(te.X)
	case *ast.SelectorExpr:
		return iscsel(te.X)
	case *ast.Ident:
		return te.Name == "C"
	}
	return false
}

func sign2rety(v string) string {
	segs := strings.Split(v, " ")
	retstr := segs[len(segs)-1]
	isptr := retstr[0] == '*'
	pos := strings.LastIndex(retstr, "/.")
	if pos > 0 {
		retstr = retstr[pos+2:]
	}
	pos = strings.LastIndex(retstr, ".")
	if pos > 0 {
		retstr = retstr[pos+1:]
	}
	if isstrty(retstr) {
		return "cxstring*"
	}
	if iseface(retstr) {
		retstr = "cxeface"
	}
	if retstr == "unsafe.Pointer" {
		return "unsafe_Pointer"
	}
	retstr = strings.TrimLeft(retstr, "*")
	return gopp.IfElseStr(isptr, retstr+"*", retstr)
}

var tmpvarno = 100

func tmpvarname() string {
	tmpvarno++
	return fmt.Sprintf("gxtv%d", tmpvarno)
}
func tmpvarname2(idx int) string {
	return fmt.Sprintf("gxtv%d", idx)
}

// idt is ast.CallExpr.Fun
func funcistypedep(idt ast.Expr) bool {
	switch te := idt.(type) {
	case *ast.Ident:
		switch te.Name {
		case "string":
			return true
		default:
			log.Println("todo", idt, reflect.TypeOf(idt))
		}
	case *ast.SelectorExpr:
		if fmt.Sprintf("%v", te.X) == "unsafe" && fmt.Sprintf("%v", te.Sel) == "Pointer" {
			return true
		}
	default:
		log.Println("todo", idt, reflect.TypeOf(idt))
	}
	return false
}

/////
type basecomp struct {
	psctx     *ParserContext
	assignkvs map[ast.Expr]ast.Expr // assign stmt left <-> right
	valnames  map[ast.Expr]ast.Expr // rvalue => lname
	strtypes  map[string]types.TypeAndValue
	closidx   map[*ast.FuncLit]*closinfo
	multirets map[*ast.FuncDecl]*ast.Ident
	deferidx  map[*ast.DeferStmt]*deferinfo
}

func newbasecomp(psctx *ParserContext) *basecomp {
	bc := &basecomp{
		assignkvs: map[ast.Expr]ast.Expr{},
		valnames:  map[ast.Expr]ast.Expr{},
		strtypes:  map[string]types.TypeAndValue{},
		closidx:   map[*ast.FuncLit]*closinfo{},
		multirets: map[*ast.FuncDecl]*ast.Ident{},
		deferidx:  map[*ast.DeferStmt]*deferinfo{}}
	bc.psctx = psctx
	bc.initbc()
	return bc
}
func (bc *basecomp) initbc() {
	psctx := bc.psctx
	for tye, tyval := range psctx.info.Types {
		bc.strtypes[bc.exprstr(tye)] = tyval
	}
}

// idt is ast.CallExpr.Fun
func (bc *basecomp) funcistype(idt ast.Expr) bool {
	tyval, ok := bc.strtypes[bc.exprstr(idt)]
	if ok {
	}
	return ok && tyval.IsType()
}

func (bc *basecomp) exprstr(e ast.Expr) string { return types.ExprString(e) }
func exprstr(e ast.Expr) string                { return types.ExprString(e) }

func (c *basecomp) exprpos(e ast.Node) token.Position {
	return exprpos(c.psctx, e)
}
func (c *basecomp) prtnode(n ast.Node) string {
	buf := bytes.NewBuffer(nil)
	printer.Fprint(buf, c.psctx.fset, n)
	return string(buf.Bytes())
}

type closinfo struct {
	idx       int
	fd        *ast.FuncDecl
	fnlit     *ast.FuncLit
	fntype    string
	fnname    string
	argtyname string
	idents    []*ast.Ident // refered identifier

}

func (bc *basecomp) newclosinfo(fd *ast.FuncDecl, fnlit *ast.FuncLit, idx int) *closinfo {
	clos := &closinfo{}
	clos.idx = idx
	clos.fd = fd
	clos.fnlit = fnlit

	funame := ""
	if fd == nil {
		funame = tmpvarname()
	} else {
		funame = fd.Name.Name
	}
	clos.fntype = fmt.Sprintf("%s_closure_type_%d", funame, idx)
	clos.fnname = fmt.Sprintf("%s_closure_%d", funame, idx)
	clos.argtyname = fmt.Sprintf("%s_closure_arg_%d", funame, idx)

	bc.fillclosidents(clos)
	return clos
}

func (bc *basecomp) fillclosidents(clos *closinfo) {
	fnlit := clos.fnlit
	myids := map[*ast.Ident]bool{}
	myids2 := map[string]bool{}
	_ = myids
	_ = myids2

	argids := map[string]bool{}
	_ = argids
	for _, prmx := range clos.fnlit.Type.Params.List {
		for _, name := range prmx.Names {
			argids[name.Name] = true
		}
	}

	// TODO proper closure ident filter
	// not arg ident
	// not self def ident
	// not other global funcs
	astutil.Apply(fnlit, func(c *astutil.Cursor) bool {
		switch te := c.Node().(type) {
		case *ast.Ident:
			gotyx := bc.psctx.info.TypeOf(te)
			switch goty := gotyx.(type) {
			case *types.Signature:
			default:
				gopp.G_USED(goty)
				if _, ok := argids[te.Name]; ok {
				} else if _, ok := myids2[te.Name]; ok {
					// log.Println("self ident", te)
				} else {
					if te.Obj == nil {
						// maybe builtin, like false/true/...
					} else {
						clos.idents = append(clos.idents, te)
					}
				}
			}
		case *ast.AssignStmt:
			for _, lvex := range te.Lhs {
				switch lve := lvex.(type) {
				case *ast.Ident:
					myids[lve] = true
					myids2[lve.Name] = true
				}
			}
		case *ast.ValueSpec:
			for _, name := range te.Names {
				myids[name] = true
				myids2[name.Name] = true
			}
		default:
			gopp.G_USED(te)
		}
		return true
	}, nil)
}

func (bc *basecomp) getclosinfo(fnlit *ast.FuncLit) *closinfo {
	closi := bc.closidx[fnlit]
	return closi
}

type deferinfo struct {
	idx    int
	defero *ast.DeferStmt
	fd     *ast.FuncDecl
	fnlit  *ast.FuncLit
}

func newdeferinfo(defero *ast.DeferStmt, idx int) *deferinfo {
	deferi := &deferinfo{}
	deferi.idx = idx
	return deferi
}
func (bc *basecomp) getdeferinfo(defero *ast.DeferStmt) *deferinfo {
	deferi := bc.deferidx[defero]
	return deferi
}

func (psctx *ParserContext) isglobal(e ast.Node) bool {
	// log.Println(e, reflect.TypeOf(e))
	switch te := e.(type) {
	case *ast.File:
		return true
	case *ast.FuncDecl:
		return false
	default:
		gopp.G_USED(te)
		return psctx.isglobal(psctx.cursors[e].Parent())
	}
}

func isglobalid(pc *ParserContext, idt *ast.Ident) bool {
	info := pc.info
	eobj := info.ObjectOf(idt)
	if eobj != nil {
		// log.Println(eobj, eobj.Pkg(), eobj.Parent())
		if eobj.Parent() != nil {
			// log.Println(eobj, eobj.Pkg(), eobj.Parent().String(), eobj.Parent().Parent())
			scope := eobj.Parent().Parent()
			if scope != nil {
				tobj := scope.Lookup("append")
				if tobj != nil && tobj.String() == "builtin append" {
					return true
				}
			}
		}
	}
	return false
}

func ispackage(pc *ParserContext, e ast.Expr) bool {
	if idt, ok := e.(*ast.Ident); ok {
		selobj := pc.info.ObjectOf(idt)
		if selobj != nil && selobj.Pkg() != nil {
			str := fmt.Sprintf("%v", selobj)
			ispkgsel :=
				strings.HasPrefix(str, "package ") && strings.Contains(str, ")")
			return ispkgsel
		}
	}
	return false
}

func reftyof(x interface{}) reflect.Type { return reflect.TypeOf(x) }

type FuncCallAttr struct {
	fnty       *types.Signature
	prmty      *types.Tuple
	isselfn    bool
	selfn      *ast.SelectorExpr
	idtfn      *ast.Ident
	ispkgsel   bool
	isrcver    bool
	iscfn      bool
	isbuiltin  bool
	isifacesel bool
	isvardic   bool
	haslval    bool
	lexpr      ast.Expr
}

type ValspecAttr struct {
	spec   *ast.ValueSpec
	validx int // used to auto calc const value

	isconst   bool
	isglobal  bool
	candefine bool // isconst && integer
	// canstatic bool // isglobal
	valty types.Type

	vp1stval ast.Expr
	vp1stty  types.Type
	vp1stidx int
}
