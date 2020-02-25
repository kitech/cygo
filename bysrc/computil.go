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
func isarrayty2(typ types.Type) bool { return isarrayty(typ.String()) }
func iseface(tystr string) bool      { return strings.HasPrefix(tystr, "interface{}") }
func iseface2(typ types.Type) bool   { return typ != nil && iseface(typ.String()) }
func isiface(tystr string) bool {
	return strings.HasPrefix(tystr, "interface{") &&
		!strings.HasPrefix(tystr, "interface{}")
}
func isiface2(typ types.Type) bool {
	if typtr, ok := typ.(*types.Pointer); ok {
		return isiface2(typtr.Elem())
	}
	tyn, ok := typ.(*types.Named)
	if !ok {
		return isiface(typ.String())
	}
	return typ != nil && isiface(tyn.Underlying().String())
}
func istypety(tystr string) bool    { return strings.HasPrefix(tystr, "type ") }
func istypety2(typ types.Type) bool { return istypety(typ.String()) }
func ischanty(tystr string) bool    { return strings.HasPrefix(tystr, "chan ") }
func ischanty2(typ types.Type) bool { return ischanty(typ.String()) }
func isvarty(tystr string) bool     { return strings.HasPrefix(tystr, "var ") }
func isvarty2(typ types.Type) bool  { return isvarty(typ.String()) }
func isstructty(tystr string) bool {
	return strings.HasPrefix(tystr, "struct{")
} // struct ???
func isstructty2(typ types.Type) bool {
	if typtr, ok := typ.(*types.Pointer); ok {
		return isstructty2(typtr.Elem())
	}
	tyn, ok := typ.(*types.Named)
	if !ok {
		return isstructty(typ.String())
	}
	return isstructty(tyn.Underlying().String())
}
func isinvalidty(tystr string) bool    { return strings.HasPrefix(tystr, "invalid ") }
func isinvalidty2(typ types.Type) bool { return isinvalidty(typ.String()) }
func isuntypedty(tystr string) bool    { return strings.HasPrefix(tystr, "untyped ") }
func isuntypedty2(typ types.Type) bool { return isuntypedty(typ.String()) }

func istuple(tystr string) bool {
	return strings.HasPrefix(tystr, "(") && strings.HasSuffix(tystr, ")")
}
func istuple2(typ types.Type) bool {
	if _, ok := typ.(*types.Tuple); ok {
		return true
	}
	return false
}
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
		return segs[0] == "builtin__error"
	}
	return segs[len(segs)-1] == "builtin_error"
}

func ispointer2(typ types.Type) bool {
	_, ok := typ.(*types.Pointer)
	return ok
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

// not include name field
func tuptyhash(ty *types.Tuple) string {
	s := ""
	for i := 0; i < ty.Len(); i++ {
		s += ty.At(i).Type().String() + "|"
	}
	return s
}

// depcreated
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
		return "builtin__cxstring3*"
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
func tmptyname() string {
	tmpvarno++
	return fmt.Sprintf("gxty%d", tmpvarno)
}
func tmptyname2(idx int) string {
	return fmt.Sprintf("gxty%d", idx)
}
func tmplabname() string {
	tmpvarno++
	return fmt.Sprintf("gxlab%d", tmpvarno)
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
	ok = ok && tyval.IsType()
	if ok {
		return true
	}
	/*
		switch te := idt.(type) {
		case *ast.Ident:
			for _, typ := range types.Typ {
				if typ.Name() == te.Name {
					// return true
				}
			}
			for _, typ := range types.AliasTyp {
				if typ.Name() == te.Name {
					//return true
				}
			}
		}
	*/
	return ok
}

func (bc *basecomp) exprstr(e ast.Expr) string { return types.ExprString(e) }
func exprstr(e ast.Expr) string                { return types.ExprString(e) }

func (c *basecomp) exprpos(e ast.Node) token.Position {
	return exprpos(c.psctx, e)
}
func (c *basecomp) prtnode(n ast.Node) string {
	defer func() {
		if p := recover(); p != nil {
			return
		}
	}()
	log.Println(n, reftyof(n))
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
	seenids := map[string]bool{}
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
						if _, ok := seenids[te.Name]; !ok {
							clos.idents = append(clos.idents, te)
							seenids[te.Name] = true
						}
					}
				}
			}
		case *ast.AssignStmt:
			if te.Tok == token.ASSIGN {
				break
			}
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
		return false // wt???
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
			ispkgsel = strings.HasPrefix(str, "package ")
			return ispkgsel
		}
	}
	return false
}

// 如果是同一个包，则返回空
func pkgpfxof(pc *ParserContext, e ast.Expr) string {
	switch te := e.(type) {
	case *ast.CallExpr:
		switch fe := te.Fun.(type) {
		case *ast.SelectorExpr:
			return pkgpfxof(pc, fe.Sel)
		case *ast.Ident:
			return pkgpfxof(pc, fe)
		}
		log.Println("noimpl", e, reftyof(e))
	case *ast.Ident:
		obj := pc.info.ObjectOf(te)
		// log.Println(e, obj.Pkg().Name(), obj.Pkg())
		if obj == nil {
			return ""
		}
		return obj.Pkg().Name() + pkgsep
	default:
		log.Println("noimpl", e, reftyof(e))
	}
	return pc.bdpkgs.Name + pkgsep
}

func reftyof(x interface{}) reflect.Type { return reflect.TypeOf(x) }

func isFuncBody(pc *ParserContext, blk *ast.BlockStmt) bool {
	blkcs, ok := pc.cursors[blk]
	if ok {
		pn := blkcs.Parent()
		if _, ok := pn.(*ast.FuncDecl); ok {
			return true
		}
		if _, ok := pn.(*ast.FuncLit); ok {
			return true
		}
	}
	return false
}

type FuncCallAttr struct {
	fnty       *types.Signature
	prmty      *types.Tuple
	isselfn    bool
	selfn      *ast.SelectorExpr
	idtfn      *ast.Ident
	ispkgsel   bool
	isrcver    bool
	iscfn      bool
	isclos     bool
	isfnvar    bool
	isbuiltin  bool
	isifacesel bool
	isvardic   bool
	haslval    bool
	haserrret  bool
	ismret     bool
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

type Annotation struct {
	original string

	// go:
	nosplit        bool
	nowritebarrier bool
	systemstack    bool
	linkname       string
	noinline       bool
	nodefer        bool

	exported   bool
	exportname string
}

func newAnnotation(cmts *ast.CommentGroup) *Annotation {
	ant := &Annotation{}
	if cmts == nil {
		return ant
	}

	for _, cmt := range cmts.List {
		cmtpfx := gopp.IfElseStr(strings.HasPrefix(cmt.Text, "//") ||
			strings.HasPrefix(cmt.Text, "/*"), "", "// ")
		cmtxt := strings.ReplaceAll(cmt.Text, "\n", "\n// ")
		ant.original += cmtpfx + cmtxt + "\n"
		lines := strings.Split(cmt.Text, "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "//go:nosplit") {
				ant.nosplit = true
			}
			if strings.HasPrefix(line, "//go:nowritebarrier") {
				ant.nowritebarrier = true
			}
			if strings.HasPrefix(line, "//go:systemstack") {
				ant.systemstack = true
			}
			if strings.HasPrefix(line, "//go:noinline") {
				ant.noinline = true
			}
			if strings.HasPrefix(line, "//go:linkname ") {
				fields := strings.Split(line, " ")
				ant.linkname = fields[1]
			}
			if strings.HasPrefix(line, "//export ") {
				fields := strings.Split(line, " ")
				ant.exported = true
				ant.exportname = fields[1]
			}
		}
	}

	return ant
}

func type2rtkind(ty types.Type) string {
	rtkind := ""
	switch ty2 := ty.(type) {
	case *types.Basic:
		rtkind = ty2.String()
	case *types.Pointer:
		rtkind = "voidptr"
	case *types.Slice:
		rtkind = "voidptr"
	case *types.Array:
		rtkind = "voidptr"
	case *types.Map:
		rtkind = "voidptr"
	case *types.Struct:
		rtkind = "struct"
	default:
		rtkind = "voidptr"
		log.Println("wtfff", ty2.String(), reftyof(ty2))
	}
	rtkind += "_metatype.kind"
	return rtkind
}

func type2rtkind2(ty types.Type) reflect.Kind {
	var rtkind reflect.Kind
	switch ty2 := ty.(type) {
	case *types.Basic:
		if ty2.Kind() == types.String {
			rtkind = reflect.Kind(reflect.String)
		} else if ty2.Kind() == types.UnsafePointer {
			rtkind = reflect.Kind(reflect.UnsafePointer)
		} else if ty2.Kind() >= types.Voidptr &&
			ty2.Kind() <= types.Wideptr {
			rtkind = reflect.Kind(uint(ty2.Kind()) + 1)
		} else {
			rtkind = reflect.Kind(ty2.Kind())
		}
	case *types.Pointer:
		rtkind = reflect.Ptr
	case *types.Slice:
		rtkind = reflect.Slice
	case *types.Array:
		rtkind = reflect.Array
	case *types.Map:
		rtkind = reflect.Map
	case *types.Struct:
		rtkind = reflect.Struct
	case *types.Named:
		return type2rtkind2(ty2.Underlying())
	default:
		rtkind = reflect.Ptr
		log.Println("wtfff", ty2.String(), reftyof(ty2))
	}
	return rtkind
}
