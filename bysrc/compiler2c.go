package main

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"gopp"
	"log"
	"path/filepath"
	"reflect"
	"runtime/debug"
	"strings"
	"unsafe"

	"github.com/thoas/go-funk"
)

func init() {
	if false {
		debug.PrintStack()
	}
}

type g2nc struct {
	*basecomp

	sb     strings.Builder
	curpkg string
	pkgo   *ast.Package

	info *types.Info
}

func (this *g2nc) genpkgs() {
	this.info = &this.psctx.info

	// pkgs order?
	for pname, pkg := range this.psctx.pkgs {
		this.geninclude_cfiles(pkg)

		pkg.Scope = ast.NewScope(nil)
		this.curpkg = pkg.Name
		this.pkgo = pkg

		this.gentypeofs(pkg)
		this.genpkg(pname, pkg)
		this.calcClosureInfo(pkg.Scope, pkg)
		this.calcDeferInfo(pkg.Scope, pkg)
		this.genGostmtTypes(pkg.Scope, pkg)
		this.genChanTypes(pkg.Scope, pkg)
		this.genMultiretTypes(pkg.Scope, pkg)
		this.genFuncs(pkg)
	}

}

const pkgsep = "__" // between pkg and type/function
const mthsep = "_"  // between type and method

func (c *g2nc) pkgpfx() string {
	pfx := ""
	if c.curpkg == "main" {
		pfx = c.curpkg + pkgsep
	} else {
		if c.psctx.pkgrename != "" {
			pfx = c.psctx.pkgrename
		} else {
			pfx = c.curpkg
		}
		pfx += pkgsep
	}
	return pfx
}

func (c *g2nc) geninclude_cfiles(pkg *ast.Package) {
	c.outf("// cfiles %d in %v", len(c.psctx.bdpkgs.CFiles), pkg.Name).outnl()
	for _, cfile := range c.psctx.bdpkgs.CFiles {
		filename := c.psctx.path + "/" + cfile
		filename2, err := filepath.Abs(filename)
		gopp.ErrPrint(err, filename)
		log.Println(filename2)
		c.outf("#include \"%s\"", filename2).outnl()
	}
}

func (this *g2nc) genpkg(name string, pkg *ast.Package) {
	log.Println("processing package", name)
	for name, f := range pkg.Files {
		this.genfile(pkg.Scope, name, f)
	}
}
func (this *g2nc) genfile(scope *ast.Scope, name string, f *ast.File) {
	log.Println("processing", name)
	/*
		for idx, cmto := range f.Comments {
			log.Println(idx, len(f.Comments), cmto.Text())
		}
	*/

	// non-func decls
	for _, d := range f.Decls {
		switch r := d.(type) {
		case *ast.FuncDecl:
		default:
			this.genDecl(scope, d)
			if r == nil {
			}
		}
	}

	// decls order?
	// for _, d := range f.Decls {
	// 	this.genDecl(scope, d)
	// }
}
func (c *g2nc) gentypeofs(pkg *ast.Package) {
	c.outf("// __typeof__ types %d in %v", len(c.psctx.csymbols), pkg.Name).outnl()
	defer c.outnl()
	for nx, _ := range c.psctx.csymbols {
		switch ne := nx.(type) {
		case *ast.CallExpr:
			fe := ne.Fun.(*ast.SelectorExpr)
			iscty := false
			if funk.Contains([]string{"int"}, fe.Sel.Name) {
				// iscty = true
				// break
			}
			if iscty {
				gopp.Assert(1 == 2, "waitdep", fe)
				c.outf("typedef __typeof__((%v)0) %v__ctype;", fe.Sel.Name, fe.Sel.Name).outnl()
			} else {
				c.outf("// typedef __typeof__((%v)(", fe.Sel.Name).outnl()
				c.outf("typedef __typeof__((%v)(", fe.Sel.Name)
				for idx, _ := range ne.Args {
					c.out("0")
					if idx == len(ne.Args)-1 {
					} else {
						c.out(",")
					}
				}
				c.outf(")) %v__ctype;", fe.Sel.Name).outnl()
			}
		case *ast.SelectorExpr:
			isstruct := strings.HasPrefix(ne.Sel.Name, "struct_")
			if isstruct {
				structname := strings.Replace(ne.Sel.Name, "_", " ", 1)
				c.outf("typedef %s %s;", structname, ne.Sel.Name).outnl()
			} else {
				c.outf("typedef __typeof__(%v) %v__const__ctype;", ne.Sel.Name, ne.Sel.Name).outnl()
			}
			c.outf("typedef __typeof__(%v) %v__ctype;", ne.Sel.Name, ne.Sel.Name).outnl()
			// TODO 字段类型
			c.outf("//typedef __typeof__(((%v*)0)->foo) %v__foo__ctype;",
				ne.Sel.Name, ne.Sel.Name).outnl()
		case *ast.Ident:
			// should be format struct_xxx.fieldxxx
			segs := strings.Split(ne.Name, ".")
			structname := strings.Replace(segs[0], "_", " ", 1)
			c.outf("typedef %s %s;", structname, segs[0]).outnl() // fix map order problem
			c.outf("typedef __typeof__(((%v*)0)->%v) %v__%v__ctype;",
				segs[0], segs[1], segs[0], segs[1]).outnl()
		default:
			log.Println("wtfff", ne)
		}
	}
}
func (c *g2nc) calcClosureInfo(scope *ast.Scope, pkg *ast.Package) {
	fds := map[*ast.FuncDecl]int{}
	for _, fnlit := range c.psctx.closures {
		fd := upfindFuncDeclAst(c.psctx, fnlit, 0)
		if fd == nil {
			// maybe global
		}
		if _, ok := fds[fd]; !ok {
			fds[fd] = 1
		} else {
			fds[fd] += 1
		}
		cnter := fds[fd]
		closi := c.newclosinfo(fd, fnlit, cnter)
		c.closidx[fnlit] = closi
	}

}
func (c *g2nc) calcDeferInfo(scope *ast.Scope, pkg *ast.Package) {
	defers := map[*ast.FuncDecl][]*ast.DeferStmt{}
	for _, defero := range c.psctx.defers {
		tmpfd := upfindFuncDeclNode(c.psctx, defero, 0)
		idx := len(defers[tmpfd])
		defers[tmpfd] = append(defers[tmpfd], defero)
		deferi := newdeferinfo(defero, idx)
		deferi.fd = tmpfd
		c.deferidx[defero] = deferi
	}
}
func (c *g2nc) genGostmtTypes(scope *ast.Scope, pkg *ast.Package) {
	c.out("// gostmt types ", fmt.Sprintf("%d", len(c.psctx.gostmts))).outnl()
	defer c.outnl()
	for idx, gostmt := range c.psctx.gostmts {
		c.outf("// %d %v %v", idx, gostmt.Call.Fun, gostmt.Call.Args).outnl()
		c.genFiberStargs(scope, gostmt.Call)
		c.outnl()
	}
}
func (c *g2nc) genChanTypes(scope *ast.Scope, pkg *ast.Package) {
	c.out("// chan types ", fmt.Sprintf("%d", len(c.psctx.chanops))).outnl()
	defer c.outnl()

	gottys := map[string]bool{}
	// te: ast.SendStmt.Chan/ast.UnaryExpr.X
	for _, te := range c.psctx.chanops {
		goty := c.info.TypeOf(te)
		gopp.Assert(ischanty2(goty), "")
		if _, ok := gottys[goty.String()]; ok {
			continue
		}
		c.genChanStargs(scope, te) // chan structure args
		gottys[goty.String()] = true
		c.outnl()
	}
}
func (c *g2nc) genMultiretTypes(scope *ast.Scope, pkg *ast.Package) {
	c.out("// multirets types ", fmt.Sprintf("%d", len(c.psctx.gostmts))).outnl()
	defer c.outnl()
	for idx, fd := range c.psctx.multirets {
		c.outf("// %d %v %v", idx, fd.Name, fd.Type.Results.NumFields()).outnl()
		c.outf("typedef struct %s_multiret_arg %s_multiret_arg", fd.Name.Name, fd.Name.Name).outfh().outnl()
		c.outf("struct %s_multiret_arg {", fd.Name.Name)

		cnter := 0
		for _, fld := range fd.Type.Results.List {
			for _, _ = range fld.Names {
				c.out(c.exprTypeName(scope, fld.Type)).outsp()
				c.out(tmpvarname2(cnter)).outfh().outnl()
				cnter++
			}
		}
		// c.genFieldList(scope, fd.Type.Results, false, true, ";", false)
		c.out("}").outfh().outnl()
		c.outnl()
	}
}

func (this *g2nc) genFuncs(pkg *ast.Package) {
	scope := pkg.Scope
	// ordered funcDeclsv
	for _, fd := range this.psctx.funcDeclsv {
		if fd == nil {
			log.Println("wtf", fd)
			continue
		}
		if fd.Name.Name == "init" {
			continue
		}
		this.genDecl(scope, fd)
	}

	this.genInitGlobvars(pkg.Scope, pkg)

	this.genInitFuncs(scope, pkg)
	if pkg.Name == "main" {
		this.genMainFunc(scope)
	}
}

func (this *g2nc) genDecl(scope *ast.Scope, d ast.Decl) {
	switch td := d.(type) {
	case *ast.FuncDecl:
		this.genPreFuncDecl(scope, td)
		this.genFuncDecl(scope, td)
		this.genPostFuncDecl(scope, td)
	case *ast.GenDecl:
		this.genGenDecl(scope, td)
	default:
		log.Println("unimplemented", reflect.TypeOf(d))
	}
}
func (c *g2nc) genPreFuncDecl(scope *ast.Scope, d *ast.FuncDecl) {
	for _, fnlit := range c.psctx.closures {
		fd2 := upfindFuncDeclAst(c.psctx, fnlit, 0)
		if fd2 != d {
			continue
		}

		closi := c.getclosinfo(fnlit)
		c.closidx[fnlit] = closi
		cnter := closi.idx

		c.outf("// %v", fnlit).outnl()
		c.out("typedef").outsp()
		c.out("struct").outsp()
		c.outf("%s_closure_arg_%d", d.Name.Name, cnter).outsp()
		c.outf("%s_closure_arg_%d", d.Name.Name, cnter).outfh().outnl()
		c.out("struct").outsp()
		c.outf("%s_closure_arg_%d", d.Name.Name, cnter).outsp()
		c.out("{").outnl()
		for _, ido := range closi.idents {
			c.out(c.exprTypeName(scope, ido)).outsp()
			c.out(ido.Name).outfh().outnl()
		}
		c.out("}").outfh().outnl()

		c.out("typedef").outsp()
		c.genFieldList(scope, fnlit.Type.Results, true, false, "", true)
		c.outf("(*%s_closure_type_%d)(", d.Name.Name, cnter)
		c.genFieldList(scope, fnlit.Type.Params, false, false, ",", false)
		c.outf("%s_closure_arg_%d*", d.Name.Name, cnter).outsp()
		c.out(")")
		c.outfh().outnl()

		c.out("static").outsp()
		c.genFieldList(scope, fnlit.Type.Results, true, false, "", true)
		c.outsp()
		c.outf("%s_closure_%d(", d.Name.Name, cnter)
		c.genFieldList(scope, fnlit.Type.Params, false, true, ",", false)
		// c.genFieldList(scope *ast.Scope, flds *ast.FieldList, keepvoid bool, withname bool, linebrk string, skiplast bool)
		c.outf("%s_closure_arg_%d*", d.Name.Name, cnter).outsp()
		c.out("clos")
		c.out(")")
		c.out("{").outnl()
		for _, ido := range closi.idents {
			c.out(c.exprTypeName(scope, ido)).outsp()
			c.out(ido.Name).outeq()
			c.out("clos", "->", ido.Name)
			c.outfh().outnl()
		}
		c.genBlockStmt(scope, fnlit.Body)
		c.out("}").outnl()
		c.outnl()
	}
}
func (c *g2nc) genPostFuncDecl(scope *ast.Scope, fd *ast.FuncDecl) {
	// gen fiber wrapper funcs
	for _, gostmt := range c.psctx.gostmts {
		// how compare called func is current func
		fe := gostmt.Call.Fun
		mat := false
		switch te := fe.(type) {
		case *ast.Ident:
			mat = te.Name == fd.Name.Name
		default:
			log.Println("todo", fe, reflect.TypeOf(fe))
		}
		if mat {
			c.genFiberStwrap(scope, gostmt.Call)
		}
	}
	ant := newAnnotation(fd.Doc)
	if ant.exported {
		c.genFieldList(scope, fd.Type.Results, true, false, "", false)
		c.outsp().out(ant.exportname).out("(")
		c.genFieldList(scope, fd.Type.Params, false, true, ",", true)
		c.out(") {").outnl()
		if fd.Type.Results != nil {
			c.out("return").outsp()
		}
		c.out(c.pkgpfx(), fd.Name.Name).out("(")
		for idx1, prm := range fd.Type.Params.List {
			for idx2, name := range prm.Names {
				c.out(name.Name)
				if idx2 == len(prm.Names)-1 && idx1 == len(fd.Type.Params.List)-1 {
				} else {
					c.out(",")
				}
			}
		}
		c.out(")").outfh().outnl()
		c.out("}").outnl()
	}
}
func (this *g2nc) genFuncDecl(scope *ast.Scope, fd *ast.FuncDecl) {
	ant := newAnnotation(fd.Doc)
	this.outf("// %v", ant.original).outnl()
	this.outf("// %v", this.exprpos(fd)).outnl()
	this.clinema(fd)
	if fd.Body == nil {
		log.Println("decl only func", fd.Name)
		if this.curpkg == "unsafe" && fd.Name.Name == "Sizeof" {
			this.out("//")
		}
		this.out("extern").outsp()
		// return
	}
	// _Cfunc_xxx
	iswcfn := iswrapcfunc(this.exprstr(fd.Name))
	ismret := fd.Type.Results.NumFields() >= 2
	if iswcfn {
		this.outf("// %v %s", fd, exprpos(this.psctx, fd)).outnl()
	}

	fdname := fd.Name.Name
	pkgpfx := this.pkgpfx()
	if ismret {
		this.outf("%s_multiret_arg*", fd.Name.Name)
	} else {
		this.genFieldList(scope, fd.Type.Results, true, false, "", false)
	}
	this.outsp()
	if fd.Recv != nil {
		recvtystr := this.exprTypeName(scope, fd.Recv.List[0].Type)
		recvtystr = strings.TrimRight(recvtystr, "*")
		this.out(recvtystr + mthsep + fd.Name.String())
	} else {
		this.out(pkgpfx + fdname)
	}
	this.out("(")
	if fd.Recv != nil {
		this.genFieldList(scope, fd.Recv, false, true, ",", true)
		if fd.Type.Params != nil && fd.Type.Params.NumFields() > 0 {
			this.out(",")
		}
	}

	this.genFieldList(scope, fd.Type.Params, false, true, ",", true)
	this.out(")").outnl()
	if iswcfn {
		this.out("{").outnl()
		if fd.Type.Results.NumFields() > 0 {
			this.out("return").outsp()
		}
		this.outf("%s(", fd.Name.Name[7:])
		for idx1, arge := range fd.Type.Params.List {
			_, isptrty := arge.Type.(*ast.StarExpr)
			for idx2, name := range arge.Names {
				this.out(gopp.IfElseStr(isptrty, "(voidptr)", ""))
				this.outf("%s", name.Name)
				if idx1 == fd.Type.Params.NumFields()-1 && idx2 == len(arge.Names)-1 {
				} else {
					this.out(",")
				}
			}
		}
		this.out(")").outfh().outnl()
		this.out("}").outnl()
	} else if fd.Body != nil {
		gendeferprep := func() {
			this.out("// int array").outnl()
			elemsz := "sizeof(int)"
			this.outf("cxarray2* deferarr = cxarray2_new(1, %v)", elemsz).outfh().outnl()
		}
		scope = ast.NewScope(scope)
		scope.Insert(ast.NewObj(ast.Fun, fd.Name.Name))
		if ismret {
			tvname := tmpvarname()
			tvidt := newIdent(tvname)
			this.multirets[fd] = tvidt
			this.out("{").outnl()
			for _, fld := range fd.Type.Results.List {
				for _, name := range fld.Names {
					this.out(this.exprTypeName(scope, fld.Type)).outsp()
					this.out(name.Name).outeq().out("{0}").outfh().outnl()
				}
			}
			this.outf("%s_multiret_arg*", fd.Name.Name).outsp().out(tvname)
			this.outeq().outsp()
			this.outf("cxmalloc(sizeof(%s_multiret_arg))", fd.Name.Name).outfh().outnl()
			gendeferprep()
			this.genBlockStmt(scope, fd.Body)
			this.out("labmret:").outnl()
			this.out("return").outsp().out(tvname).outfh().outnl()
			this.out("}").outnl()
		} else {
			this.out("{").outnl()
			gendeferprep()
			this.genBlockStmt(scope, fd.Body)
			this.out("}").outnl()
		}
	} else {
		this.outfh()
	}
	this.outnl()
}
func (c *g2nc) genMainFunc(scope *ast.Scope) {
	c.out("int main(int argc, char**argv) {").outnl()
	c.out("cxrt_init_env(argc, argv)").outfh().outnl()
	c.out("// TODO arguments populate").outnl()
	c.out("// globvars populate").outnl()
	c.out("extern void cxall_globvars_init()").outfh().outnl()
	c.out("cxall_globvars_init()").outfh().outnl()
	c.outf("%sglobvars_init()", c.pkgpfx()).outfh().outnl()
	c.out("extern void cxall_pkginit()").outfh().outnl()
	c.out("cxall_pkginit()").outfh().outnl()
	c.out("// all func init()").outnl()
	c.outf("%spkginit()", c.pkgpfx()).outfh().outnl()
	c.outf("main%smain()", pkgsep).outfh().outnl()
	c.out("return 0").outfh().outnl()
	c.out("}").outnl()
}

// per package
func (c *g2nc) genInitFuncs(scope *ast.Scope, pkg *ast.Package) {
	for idx, fd := range c.psctx.initFuncs {
		c.outf("// %s", c.exprpos(fd).String()).outnl()
		c.out("static").outsp()
		c.outf("void %spkginit_%d()", c.pkgpfx(), idx)
		c.genBlockStmt(scope, fd.Body)
	}
	c.outf("void %spkginit(){", c.pkgpfx()).outnl()
	for idx, _ := range c.psctx.initFuncs {
		c.outf("%spkginit_%d()", c.pkgpfx(), idx).outfh().outnl()
	}
	c.out("}").outnl().outnl()
}

// all packages
func (c *g2nc) genCallPkgGlobvarsInits(pkgs []string) {
	c.out("void cxall_globvars_init() {").outnl()
	last := pkgs[len(pkgs)-1] // builtin
	pkgs = append([]string{last}, pkgs[:len(pkgs)-1]...)
	for _, pkg := range pkgs {
		c.outf("  %s%sglobvars_init()", pkg, pkgsep).outfh().outnl()
	}
	c.out("}").outnl()
}
func (c *g2nc) genCallPkgInits(pkgs []string) {
	c.out("void cxall_pkginit() {").outnl()
	last := pkgs[len(pkgs)-1] // builtin
	pkgs = append([]string{last}, pkgs[:len(pkgs)-1]...)
	for _, pkg := range pkgs {
		c.outf("  %s%spkginit()", pkg, pkgsep).outfh().outnl()
	}
	c.out("}").outnl()
}

func (this *g2nc) genBlockStmt(scope *ast.Scope, stmt *ast.BlockStmt) {
	this.out("{").outnl()
	scope = ast.NewScope(scope)

	tailreturn := false
	var tailstmt ast.Stmt
	for idx, s := range stmt.List {
		this.genStmt(scope, s, idx)
		if idx == len(stmt.List)-1 {
			_, tailreturn = s.(*ast.ReturnStmt)
			if !tailreturn {
				tailstmt = s
			}
		}
	}

	pcn := this.psctx.cursors[stmt].Parent()
	_, isfuncblk := pcn.(*ast.FuncDecl)
	if isfuncblk {
		if !tailreturn {
			this.genDeferStmt(scope, tailstmt)
		}
	}
	this.out("}").outnl()
}

// clause index?
func (this *g2nc) genStmt(scope *ast.Scope, stmt ast.Stmt, idx int) {
	// log.Println(stmt, reflect.TypeOf(stmt))
	if stmt != nil {
		posinfo := this.exprpos(stmt).String()
		fields := strings.Split(posinfo, ":")
		if len(fields) > 1 {
			// this.outf("#line %s \"%s\"", fields[1], fields[0]).outnl()
			this.out("// ", posinfo).outnl()
		} else {
			this.out("// ", posinfo).outnl()
		}
		stmtstr := this.prtnode(stmt)
		if !strings.ContainsAny(strings.TrimSpace(stmtstr), "\n") {
			this.outf("// %s", stmtstr).outnl()
		}
		this.genStmtTmps(scope, stmt)
	}
	defer this.outnl()

	addfh := true
	switch t := stmt.(type) {
	case *ast.AssignStmt:
		this.genAssignStmt(scope, t)
	case *ast.ExprStmt:
		this.genExpr(scope, t.X)
	case *ast.GoStmt:
		this.genGoStmt(scope, t)
	case *ast.ForStmt:
		this.genForStmt(scope, t)
	case *ast.RangeStmt:
		this.genRangeStmt(scope, t)
	case *ast.IncDecStmt:
		this.genIncDecStmt(scope, t)
	case *ast.BranchStmt:
		this.genBranchStmt(scope, t)
	case *ast.DeclStmt:
		this.genDeclStmt(scope, t)
	case *ast.IfStmt:
		this.genIfStmt(scope, t)
	case *ast.BlockStmt:
		this.genBlockStmt(scope, t)
	case *ast.SwitchStmt:
		this.genSwitchStmt(scope, t)
	case *ast.CaseClause:
		// addfh = false
		this.genCaseClause(scope, t, idx)
	case *ast.SendStmt:
		this.genSendStmt(scope, t)
	case *ast.ReturnStmt:
		this.genReturnStmt(scope, t)
	case *ast.DeferStmt:
		this.genDeferStmtSet(scope, t)
	default:
		if stmt == nil { // empty block {}
		} else {
			log.Println("unknown", reflect.TypeOf(stmt), t)
		}
	}
	if addfh {
		this.outfh().outnl()
	}
}
func (c *g2nc) genStmtTmps(scope *ast.Scope, stmt ast.Stmt) {
	if nodes, ok := c.psctx.tmpvars[stmt]; ok {
		c.outf("// temporary vars %v", len(nodes)).outnl()
		for _, n := range nodes {
			switch en := n.(type) {
			case ast.Stmt:
				c.genStmt(scope, en, 0)
			case *ast.ValueSpec:
				c.genValueSpec(scope, en, 0)
			}
		}
	}
}

func (c *g2nc) genAssignStmt(scope *ast.Scope, s *ast.AssignStmt) {
	// log.Println(s.Tok.String(), s.Tok.Precedence(), s.Tok.IsOperator(), s.Tok.IsLiteral(), s.Lhs)
	for i := 0; i < len(s.Rhs); i++ {
		c.valnames[s.Rhs[i]] = s.Lhs[i]
		switch te := s.Lhs[i].(type) {
		case *ast.Ident:
			obj := ast.NewObj(ast.Var, te.Name)
			obj.Data = s.Rhs[i]
			scope.Insert(obj)
		}
		var ischrv = false
		var chexpr ast.Expr
		switch e := s.Rhs[i].(type) {
		case *ast.UnaryExpr:
			if e.Op.String() == "<-" {
				ischrv = true
				chexpr = e.X
			}
		default:
			// log.Println("unknown", reflect.TypeOf(e))
		}
		_, isidxas := s.Lhs[i].(*ast.IndexExpr)

		if ischrv {
			if s.Tok.String() == ":=" {
				c.out(c.chanElemTypeName(chexpr, false)).outsp()
				c.genExpr(scope, s.Lhs[i])
				// c.outfh().outnl()
			}

			var ns = putscope(scope, ast.Var, "varname", s.Lhs[i])
			c.genExpr(ns, s.Rhs[i])
		} else if isidxas {
			if s.Tok.String() == ":=" {
				c.out(c.exprTypeName(scope, s.Rhs[i])).outsp()
			}
			var ns = putscope(scope, ast.Var, "varval", s.Rhs[i])
			c.genExpr(ns, s.Lhs[i])
		} else if istuple(c.exprTypeName(scope, s.Rhs[i])) {
			tvname := tmpvarname()
			var ns = putscope(scope, ast.Var, "varname", newIdent(tvname))
			c.out(c.exprTypeName(scope, s.Rhs[i]))
			c.out(tvname).outeq()
			c.genExpr(ns, s.Rhs[i])
			c.outfh().outnl()
			for idx, te := range s.Lhs {
				c.out(c.exprTypeName(scope, te)).outsp()
				switch xe := te.(type) {
				case *ast.Ident:
					c.out(xe.Name).outeq()
				case *ast.SelectorExpr:
					c.genExpr(scope, xe)
				default:
					log.Panicln(te, reftyof(te), exprstr(te))
				}
				c.out(tvname).out("->").out(tmpvarname2(idx)).outfh().outnl()
			}
			c.outf("cxfree(%s)", tvname).outfh().outnl()
		} else if iserrorty2(c.info.TypeOf(s.Lhs[i])) {
			c.out("error*").outsp()
			c.genExpr(scope, s.Lhs[i])
			c.outeq()
			if c.info.TypeOf(s.Rhs[i]) == c.info.TypeOf(s.Lhs[i]) {
				c.genExpr(scope, s.Rhs[i])
			} else {
				c.out("error_new_zero()").outfh().outnl()
				c.genExpr(scope, s.Lhs[i])
				c.outf("->data").outeq()
				c.genExpr(scope, s.Rhs[i])
				c.outfh().outnl()
				c.genExpr(scope, s.Lhs[i])
				c.outf("->Error").outeq()
				c.outf("%s_Error", strings.Trim(c.exprTypeName(scope, s.Rhs[i]), "*"))
			}
		} else {
			if s.Tok == token.DEFINE {
				c.out(c.exprTypeName(scope, s.Rhs[i])).outsp()
			}
			c.genExpr(scope, s.Lhs[i])

			goty := c.info.TypeOf(s.Rhs[i])
			var ns = putscope(scope, ast.Var, "varname", s.Lhs[i])
			if s.Tok == token.DEFINE {
				c.outeq()
			} else if s.Tok == token.AND_NOT_ASSIGN {
				c.out(s.Tok.String()) // todo
			} else {
				if isstrty2(goty) && s.Tok == token.ADD_ASSIGN {
					c.outeq()
				} else {
					c.out(s.Tok.String())
				}
			}
			if isstrty2(goty) && s.Tok == token.ADD_ASSIGN {
				c.out("cxstring_add(")
				c.genExpr(scope, s.Lhs[i])
				c.out(",")
			}
			c.genExpr(ns, s.Rhs[i])
			if isstrty2(goty) && s.Tok == token.ADD_ASSIGN {
				c.out(")")
			}
			// c.outfh().outnl()
		}
	}

}
func (this *g2nc) genGoStmt(scope *ast.Scope, stmt *ast.GoStmt) {
	// calleename := stmt.Call.Fun.(*ast.Ident).Name
	// this.genCallExpr(scope, stmt.Call)
	// define function in function in c?
	// this.genFiberStargs(scope, stmt.Call)
	// this.genFiberStwrap(scope, stmt.Call)
	this.genFiberWcall(scope, stmt.Call)
}
func (c *g2nc) genFiberStargs(scope *ast.Scope, e *ast.CallExpr) {
	var funame string
	switch te := e.Fun.(type) {
	case *ast.Ident:
		funame = e.Fun.(*ast.Ident).Name
		if _, ok := c.psctx.grstargs[funame]; ok {
			return
		}
	case *ast.FuncLit:
		closi := c.getclosinfo(te)
		funame = closi.fnname
	default:
		log.Println("todo", e, reflect.TypeOf(e.Fun))
	}

	c.out("typedef struct {")
	for idx, ae := range e.Args {
		fldname := fmt.Sprintf("a%d", idx)
		fldtype := c.exprTypeName(scope, ae)
		// log.Println(funame, fldtype, fldname, reflect.TypeOf(ae))
		c.out(fldtype).outsp().out(fldname).outfh().outnl()
	}
	c.out("}", funame+"_fiber_args").outfh().outnl()
}
func (c *g2nc) genFiberStwrap(scope *ast.Scope, e *ast.CallExpr) {
	funame := e.Fun.(*ast.Ident).Name
	if _, ok := c.psctx.grstargs[funame]; ok {
		return
	}
	c.psctx.grstargs[funame] = true

	fnobj := c.info.ObjectOf(e.Fun.(*ast.Ident))
	pkgo := fnobj.Pkg()

	stname := funame + "_fiber_args"
	c.out("static").outsp()
	c.out("void").outsp()
	c.out(gopp.IfElseStr(pkgo == nil, "", pkgo.Name()+"_"))
	c.out(funame+"_fiber", "(voidptr vpargs)").outnl()
	c.out("{").outnl()
	c.out(stname, "*args = (", stname, "*)vpargs").outfh().outnl()
	c.out(gopp.IfElseStr(pkgo == nil, "", pkgo.Name()+pkgsep))
	c.out(funame, "(")
	for idx, _ := range e.Args {
		fldname := fmt.Sprintf("args->a%d", idx)
		c.out(fldname)
		c.out(gopp.IfElseStr(idx == len(e.Args)-1, "", ","))
	}
	c.out(")").outfh().outnl()
	c.out("}").outnl().outnl()
}
func (c *g2nc) genFiberWcall(scope *ast.Scope, e *ast.CallExpr) {
	funame := e.Fun.(*ast.Ident).Name
	wfname := funame + "_fiber"
	stname := funame + "_fiber_args"

	fnobj := c.info.ObjectOf(e.Fun.(*ast.Ident))
	pkgo := fnobj.Pkg()

	c.out("// gogorun", funame).outnl()
	c.out("{")
	c.outf("%s* args = (%s*)cxmalloc(sizeof(%s))", stname, stname, stname).outfh().outnl()
	for idx, arg := range e.Args {
		c.outf("args->a%d", idx).outeq()
		c.genExpr(scope, arg)
		c.outfh().outnl()
	}
	pkgpfx := gopp.IfElseStr(pkgo == nil, "", pkgo.Name())
	c.outf("cxrt_fiber_post(%s_%s, args)", pkgpfx, wfname).outfh().outnl()
	c.out("}").outnl()
}

func (c *g2nc) genForStmt(scope *ast.Scope, s *ast.ForStmt) {
	isefor := s.Init == nil && s.Cond == nil && s.Post == nil // for {}
	tmpv := tmpvarname()
	c.outf("int %s = 0", tmpv).outfh().outnl()
	c.out("for (")
	c.genStmt(scope, s.Init, 0)
	// c.out(";") // TODO ast.AssignStmt has put ;
	if isefor {
		// c.out(";")
	}
	// c.genExpr(scope, s.Cond)
	c.out(";")
	c.out(")")

	c.out("{")
	c.outf("if (%s>0) {", tmpv).outnl()
	c.genStmt(scope, s.Post, 2)
	c.outf("} else { %s = 1; }", tmpv)
	c.outf("if (")
	if s.Cond == nil {
		c.out("1")
	} else {
		c.genExpr(scope, s.Cond)
	}
	c.outf(") {\n /* goon */\n } else {break;}").outnl()

	c.genBlockStmt(scope, s.Body)
	// c.genStmt(scope, s.Post, 2) // Post move to real post, to resolve ';' problem
	c.out("// TODO gc safepoint code").outnl()
	c.out("}")
}
func (c *g2nc) genRangeStmt(scope *ast.Scope, s *ast.RangeStmt) {
	varty := c.info.TypeOf(s.X)
	// log.Println(varty, reflect.TypeOf(varty))
	switch be := varty.(type) {
	case *types.Map:
		keytystr := c.exprTypeName(scope, s.Key)
		valtystr := c.exprTypeName(scope, s.Value)

		c.out("{").outnl()
		c.out("  HashTableIter htiter").outfh().outnl()
		c.out("  hashtable_iter_init(&htiter, ")
		c.genExpr(scope, s.X)
		c.out(")").outfh().outnl()
		c.out("  TableEntry *entry").outfh().outnl()
		c.out("  while (hashtable_iter_next(&htiter, &entry) != CC_ITER_END) {").outnl()
		c.outf("    %s %v = entry->key", keytystr, s.Key).outfh().outnl()
		c.outf("    %s %v = entry->value", valtystr, s.Value).outfh().outnl()
		c.genBlockStmt(scope, s.Body)
		c.out("  }").outnl()
		c.out("// TODO gc safepoint code").outnl()
		c.out("}").outnl()
	case *types.Slice:
		keyidstr := fmt.Sprintf("%v", s.Key)
		keyidstr = gopp.IfElseStr(keyidstr == "_", "idx", keyidstr)

		c.out("{").outnl()
		c.outf("  for (int %s = 0; %s < cxarray2_size(%v); %s++) {",
			keyidstr, keyidstr, s.X, keyidstr).outnl()
		if s.Value != nil {
			valtystr := c.exprTypeName(scope, s.Value)
			c.outf("     %s %v = {0}", valtystr, s.Value).outfh().outnl()
			var tmpvar = tmpvarname()
			c.outf("    voidptr %s = {0}", tmpvar).outfh().outnl()
			c.outf("    %v = *cxarray2_get_at(%v, %s)", tmpvar, s.X, keyidstr).outfh().outnl()
		}
		c.genBlockStmt(scope, s.Body)
		c.out("  }").outnl()
		c.out("// TODO gc safepoint code").outnl()
		c.out("}").outnl()
		if be == nil {
		}
	// TODO Array/String
	default:
		if isstrty2(varty) {
			keyidstr := fmt.Sprintf("%v", s.Key)
			keyidstr = gopp.IfElseStr(keyidstr == "_", "idx", keyidstr)
			valtystr := c.exprTypeName(scope, s.Value)

			c.out("{").outnl()
			c.outf("  for (int %s = 0; %s < (%v)->len; %s++) {",
				keyidstr, keyidstr, s.X, keyidstr).outnl()
			c.outf("     %s %v = {0}", valtystr, s.Value).outfh().outnl()
			c.outf("    %v = (%v->ptr)[%s]", s.Value, s.X, keyidstr).outfh().outnl()
			c.genBlockStmt(scope, s.Body)
			c.out("  }").outnl()
			c.out("// TODO gc safepoint code").outnl()
			c.out("}").outnl()
		} else {
			log.Println("todo", s.Key, s.Value, s.X, varty)
		}
	}
}
func (c *g2nc) genIncDecStmt(scope *ast.Scope, s *ast.IncDecStmt) {
	c.genExpr(scope, s.X)
	if s.Tok.IsOperator() {
		c.out(s.Tok.String())
	}
}
func (c *g2nc) genBranchStmt(scope *ast.Scope, s *ast.BranchStmt) {
	if s.Tok == token.FALLTHROUGH {
		c.out("gxtvnextcase = 1; break")
	} else {
		c.out(s.Tok.String())
		if s.Label != nil {
			c.out(s.Label.Name)
		}
	}
}
func (c *g2nc) genDeclStmt(scope *ast.Scope, s *ast.DeclStmt) {
	c.genDecl(scope, s.Decl)
}

func (c *g2nc) genIfStmt(scope *ast.Scope, s *ast.IfStmt) {
	if s.Init != nil {
		c.genStmt(scope, s.Init, 0)
	}
	c.out("if (")
	c.genExpr(scope, s.Cond)
	c.out(")")
	c.genBlockStmt(scope, s.Body)
	if s.Else != nil {
		c.out("else").outsp()
		c.genStmt(scope, s.Else, 0)
	}
}
func (c *g2nc) genSwitchStmt(scope *ast.Scope, s *ast.SwitchStmt) {
	tagty := c.info.TypeOf(s.Tag)
	if tagty == nil {
		log.Println(tagty, c.exprpos(s))
	} else {
		log.Println(tagty, reflect.TypeOf(tagty), reflect.TypeOf(tagty.Underlying()))
	}
	switch tty := tagty.(type) {
	case *types.Basic:
		switch tty.Kind() {
		case types.Int:
			// c.genSwitchStmtNum(scope, s)
			c.genSwitchStmtAsIf(scope, s)
		default:
			log.Println("unknown", tagty, reflect.TypeOf(tagty))
		}
	default:
		if tagty == nil {
			c.genSwitchStmtIf(scope, s)
		} else {
			log.Println("unknown", tagty, reflect.TypeOf(tagty))
		}
	}

}
func (c *g2nc) genSwitchStmtNum(scope *ast.Scope, s *ast.SwitchStmt) {
	c.out("switch (")
	c.genExpr(scope, s.Tag)
	c.out(")")
	c.genBlockStmt(scope, s.Body)
}
func (c *g2nc) genSwitchStmtStr(scope *ast.Scope, s *ast.SwitchStmt) {
	log.Println(s.Tag)
}
func (c *g2nc) genCaseClause(scope *ast.Scope, s *ast.CaseClause, idx int) {
	log.Println(s.List, s.Body)
	if len(s.List) == 0 {
		// default
		c.out("default:").outnl()
		for idx, s_ := range s.Body {
			c.genStmt(scope, s_, idx)
		}
	} else {
		switch s.List[0].(type) {
		case *ast.BinaryExpr:
			c.genCaseClauseIf(scope, s, idx)
			return
		}

		// TODO precheck if have fallthrough
		for idx, ce := range s.List {
			c.out("case").outsp()
			c.genExpr(scope, ce)
			c.out(":").outnl()
			for idx2, be := range s.Body {
				c.genStmt(scope, be, idx2)
			}
			c.out("break").outfh().outnl()
			gopp.G_USED(idx)
		}
	}
}
func (c *g2nc) genSwitchStmtIf(scope *ast.Scope, s *ast.SwitchStmt) {
	log.Println(s.Tag, s.Body == nil)
	c.outf("// %v", reflect.TypeOf(s)).outnl()
	c.genBlockStmt(scope, s.Body)
}
func (c *g2nc) genCaseClauseIf(scope *ast.Scope, s *ast.CaseClause, idx int) {
	log.Println(s.List, s.Body)
	for _, expr := range s.List {
		log.Println(expr, reflect.TypeOf(expr))
	}
	if len(s.List) == 0 {
		// default
		c.out("default:").outnl()
		for idx, s_ := range s.Body {
			c.genStmt(scope, s_, idx)
		}
	} else {
		// TODO precheck if have fallthrough
		c.out("//", gopp.IfElseStr(idx > 0, "else", "")).outnl()
		c.out("if (")
		for idx, ce := range s.List {
			c.out("(")
			c.genExpr(scope, ce)
			c.out(")")
			if idx < len(s.List)-1 {
				c.out("||").outnl()
			}
			gopp.G_USED(idx)
		}
		c.out(") {").outnl()
		for idx, be := range s.Body {
			c.genStmt(scope, be, idx)
		}
		c.out("}").outnl()
	}
}

// TODO c switch too weak, use c if stmt
func (c *g2nc) genSwitchStmtAsIf(scope *ast.Scope, s *ast.SwitchStmt) {
	c.out("{ // switch asif").outnl()
	if s.Init != nil {
		c.genStmt(scope, s.Init, 0)
	}
	lst := s.Body.List
	tmplabs := []string{}
	for range lst {
		tmplabs = append(tmplabs, tmpvarname())
	}
	for idx, stmtx := range lst {
		stmt := stmtx.(*ast.CaseClause)
		log.Println(stmt, reftyof(stmt))
		c.outf(gopp.IfElseStr(idx > 0, "else", "")).outsp()
		c.outf("// %v", exprpos(c.psctx, stmt)).outnl()
		c.outf("if (")
		for idx2, exprx := range stmt.List {
			c.genExpr(scope, s.Tag)
			c.out(token.EQL.String())
			c.genExpr(scope, exprx)
			c.out(gopp.IfElseStr(idx2 < len(stmt.List)-1, "||", ""))
		}
		if len(stmt.List) == 0 { //default
			c.out("1")
		}
		c.outf(") {").outnl()
		c.outf("%s:", tmplabs[idx]).outfh().outnl()
		c.outf("int gxtvnextcase = 0").outfh().outnl()
		c.outf("do {").outnl()
		for idx2, s2 := range stmt.Body {
			c.genStmt(scope, s2, idx2)
		}
		c.outf("} while(0)").outfh().outnl()
		c.outf("if (gxtvnextcase==1) {").outnl()
		if idx >= len(lst)-1 {
			c.outf("// goto %s+1", tmplabs[idx]).outnl()
		} else {
			c.outf("goto %s", tmplabs[idx+1]).outfh().outnl()
		}
		c.outf("}").outnl()
		c.outf("}").outnl()
	}
	c.out("}").outnl()
}

func (c *g2nc) genCallExpr(scope *ast.Scope, te *ast.CallExpr) {
	// log.Println(te, te.Fun, reflect.TypeOf(te.Fun))
	scope = putscope(scope, ast.Fun, "infncall", te.Fun)
	fca := c.getCallExprAttr(scope, te)
	switch be := te.Fun.(type) {
	case *ast.Ident:
		funame := be.Name
		if funame == "make" {
			c.genCallExprMake(scope, te)
		} else if funame == "len" {
			c.genCallExprLen(scope, te)
		} else if funame == "cap" {
			c.genCallExprLen(scope, te)
			// panic("not supported " + funame)
		} else if funame == "append" {
			c.genCallExprAppend(scope, te)
		} else if funame == "delete" {
			c.genCallExprDelete(scope, te)
		} else if funame == "println" {
			c.genCallExprPrintln(scope, te)
		} else if c.funcistype(be) {
			c.genTypeCtor(scope, te)
		} else {
			var upfindsym func(s *ast.Scope, id *ast.Ident, lvl int) interface{}
			upfindsym = func(s *ast.Scope, id *ast.Ident, lvl int) interface{} {
				if s == nil {
					return nil
				}
				obj := s.Lookup(id.Name)
				if obj != nil {
					switch id2 := obj.Data.(type) {
					case *ast.Ident:
						return upfindsym(s, id2, 0)
					}
					return obj.Data
				}
				return upfindsym(s.Outer, id, lvl+1)
			}
			isclos := false
			var fnlit *ast.FuncLit
			gotyx := c.info.TypeOf(te.Fun)
			switch gotyx.(type) {
			case *types.Signature:
				symve := upfindsym(scope, be, 0)
				isclos = symve != nil
				if isclos {
					fnlit = symve.(*ast.FuncLit)
				}
			}
			if isclos {
				c.genCallExprClosure(scope, te, fnlit)
			} else if fca.isvardic {
				c.genCallExprVaridic(scope, te)
			} else {
				c.genCallExprNorm(scope, te)
			}
		}
	case *ast.SelectorExpr:
		if fca.isbuiltin &&
			funk.Contains([]string{"sizeof", "alignof", "offsetof", "assert"},
				fca.selfn.Sel.Name) {
			selname := fca.selfn.Sel.Name
			selname = gopp.IfElseStr(selname == "alignof", "_Alignof", selname)
			c.outf("%s(", selname)
			for idx, _ := range te.Args {
				c.genExpr(scope, te.Args[idx])
				c.out(gopp.IfElseStr(idx == 0 && len(te.Args) > 1, ",", ""))
			}
			c.out(")")
			break
		}
		if c.funcistype(te.Fun) {
			c.genTypeCtor(scope, te)
		} else if fca.isvardic {
			c.genCallExprVaridic(scope, te)
		} else {
			c.genCallExprNorm(scope, te)
		}
	case *ast.ArrayType:
		c.genTypeCtor(scope, te)
	case *ast.ParenExpr:
		c.out("(")
		log.Println(be.X, reflect.TypeOf(be.X))
		c.genExpr(scope, be.X)
		c.out(")")
		c.out("(")
		for idx, arge := range te.Args {
			c.genExpr(scope, arge)
			if idx < len(te.Args)-1 {
				c.out(",")
			}
		}
		c.out(")")
	case *ast.FuncLit:
		c.genCallExprClosure(scope, te, be)
	default:
		log.Println("todo", be, reflect.TypeOf(be))
	}
}
func (c *g2nc) genCallExprMake(scope *ast.Scope, te *ast.CallExpr) {
	log.Println("CallExpr", te.Fun)
	itep := te.Args[0]
	var lenep ast.Expr
	if len(te.Args) > 1 {
		lenep = te.Args[1]
	}

	log.Println(reflect.TypeOf(itep))
	switch ity := itep.(type) {
	case *ast.ChanType:
		log.Println("elemty", reflect.TypeOf(ity.Value), c.info.TypeOf(ity.Value))
		elemtyx := c.info.TypeOf(ity.Value)
		log.Println(elemtyx, reflect.TypeOf(elemtyx))
		switch elemty := elemtyx.(type) {
		case *types.Basic:
			switch elemty.Kind() {
			case types.Int:
				log.Println("it's chan, and elem int", lenep)
			default:
				log.Println("unknown", elemtyx, elemty)
			}
		default:
			log.Println("unknown", elemtyx, elemty)
		}
		c.out("cxrt_chan_new(")
		if lenep == nil {
			c.out("0")
		} else {
			c.genExpr(scope, lenep)
		}
		c.out(")")
	case *ast.ArrayType:
		gopp.Assert(len(te.Args) > 0, "wtfff", len(te.Args))
		acap := "1"
		if len(te.Args) > 1 {
			switch elme := te.Args[1].(type) {
			case *ast.BasicLit:
				acap = elme.Value
			case *ast.Ident:
				acap = elme.Name
			default:
				log.Panicln(elme, reftyof(elme), exprstr(te), exprpos(c.psctx, te))
			}
		}
		elemtya := te.Args[0].(*ast.ArrayType).Elt
		log.Println(te.Args[0], reftyof(te.Args[0]), elemtya, reftyof(elemtya))
		elemtyt := c.info.TypeOf(elemtya)
		var elemsz interface{} = (&types.StdSizes{}).Sizeof(elemtyt)
		elemsz = gopp.IfElse(elemsz == uintptr(0), "sizeof(voidptr)", elemsz)
		c.outf("cxarray2_new(%v, %v)", acap, elemsz)
	default:
		log.Println("unknown", itep, ity, lenep)
	}
}
func (c *g2nc) genCallExprLen(scope *ast.Scope, te *ast.CallExpr) {
	arg0 := te.Args[0]
	argty := c.info.TypeOf(arg0)
	if ismapty(argty.String()) {
		switch be := arg0.(type) {
		case *ast.Ident:
			c.outf("hashtable_size(%s)", be.Name)
		case *ast.SelectorExpr:
			c.out("hashtable_size(")
			c.genExpr(scope, be.X)
			c.out("->")
			c.genExpr(scope, be.Sel)
			c.out(")")
		default:
			log.Println("todo", reflect.TypeOf(arg0))
		}
	} else if isstrty(argty.String()) {
		c.out("cxstring_len(")
		c.genExpr(scope, arg0)
		c.out(")")
	} else if isslicety(argty.String()) || isarrayty(argty.String()) {
		funame := te.Fun.(*ast.Ident).Name
		if funame == "len" {
			c.outf("cxarray2_size(")
			c.genExpr(scope, arg0)
			c.out(")")
		} else if funame == "cap" {
			c.out("cxarray2_capacity(")
			c.genExpr(scope, arg0)
			c.out(")")
		} else {
			panic(funame)
		}
	} else {
		log.Println("todo", te.Args, argty)
	}
}
func (c *g2nc) genCallExprAppend(scope *ast.Scope, te *ast.CallExpr) {
	arg0 := te.Args[0]
	argty := c.info.TypeOf(arg0)
	if ismapty(argty.String()) {
		panic(argty.String())
		switch be := arg0.(type) {
		case *ast.Ident:
			c.outf("hashtable_size(%s)", be.Name)
		case *ast.SelectorExpr:
			c.out("hashtable_size(")
			c.genExpr(scope, be.X)
			c.out("->")
			c.genExpr(scope, be.Sel)
			c.out(")")
		default:
			log.Println("todo", reflect.TypeOf(arg0))
		}
	} else if isstrty(argty.String()) {
		panic(argty.String())
		c.out("cxstring_len(")
		c.genExpr(scope, arg0)
		c.out(")")
	} else if isslicety(argty.String()) || isarrayty(argty.String()) {
		// funame := te.Fun.(*ast.Ident).Name
		for idx := 1; idx < len(te.Args); idx++ {
			ae := te.Args[idx]
			if idx > 1 {
				c.genExpr(scope, arg0)
				c.outeq()
			}
			c.outf("cxarray2_append(")
			c.genExpr(scope, arg0)
			c.out(", (voidptr)&")
			c.genExpr(scope, ae)
			c.out(")").outfh().outnl()
		}

	} else {
		log.Println("todo", te.Args, argty)
	}
}
func (c *g2nc) genCallExprDelete(scope *ast.Scope, te *ast.CallExpr) {
	arg0 := te.Args[0]
	arg1 := te.Args[1]
	argty := c.info.TypeOf(arg0)
	if ismapty(argty.String()) {
		keystr := ""
		switch te := arg1.(type) {
		case *ast.BasicLit:
			switch te.Kind {
			case token.STRING:
				keystr = fmt.Sprintf("cxhashtable_hash_str(%s)", te.Value)
			default:
				log.Println("todo", te.Kind)
			}
		case *ast.Ident:
			keystr = c.exprstr(arg1)
		default:
			log.Println("todo", reflect.TypeOf(arg1), arg1, c.exprstr(arg1), c.exprpos(arg0))
		}
		c.outf("hashtable_remove(")
		c.genExpr(scope, arg0)
		c.outf(", (voidptr)(uintptr_t)%s, 0)", keystr).outfh().outnl()
	} else {
		log.Println("todo", te.Args, argty)
	}
}
func (c *g2nc) genCallExprPrintln(scope *ast.Scope, te *ast.CallExpr) {
	tmpnames := make([]string, len(te.Args))
	for idx, e1 := range te.Args {
		tety := c.info.TypeOf(e1)
		if isstrty2(tety) {
			switch tety.(type) {
			case *types.Basic:
				tname := tmpvarname()
				c.outf("cxstring* %s = ", tname)
				c.genExpr(scope, e1)
				c.outfh().outnl()
				tmpnames[idx] = tname
			}
		}
	}
	// c.genExpr(scope, te.Fun)
	c.out("println2")
	c.out("(__FILE__, __LINE__, __func__")
	c.out(gopp.IfElseStr(len(te.Args) > 0, ",", "")).outnl()
	if len(te.Args) > 0 {
		var tyfmts []string
		for _, e1 := range te.Args {
			tyfmt := c.exprTypeFmt(scope, e1)
			tyfmts = append(tyfmts, "%"+tyfmt)
		}
		c.out(fmt.Sprintf(`"%s"`, strings.Join(tyfmts, " ")))
		c.out(", ")
	}
	for idx, e1 := range te.Args {
		tety := c.info.TypeOf(e1)
		if isstrty2(tety) {
			c.outf("(%s)->len,", tmpnames[idx])
			c.outf("(%s)->ptr", tmpnames[idx])
		} else if iseface2(tety) {
			c.genExpr(scope, e1)
			c.out(".data")
		} else {
			c.genExpr(scope, e1)
		}
		c.out(gopp.IfElseStr(idx == len(te.Args)-1, "", ", "))
	}
	c.out(")")

	// check if real need, ;\n
	cs := c.psctx.cursors[te]
	if cs.Name() != "Args" {
		// c.outfh().outnl()
	}
}
func (c *g2nc) getCallExprAttr(scope *ast.Scope, te *ast.CallExpr) *FuncCallAttr {
	fca := &FuncCallAttr{}
	fca.selfn, fca.isselfn = te.Fun.(*ast.SelectorExpr)
	if fca.isselfn {
		// var selidt *ast.Ident
		// selidt, isidt = selfn.X.(*ast.Ident)
		// iscfn = isidt && selidt.Name == "C"
		if iscsel(te.Fun) {
			// selidt = selfn.Sel.(*ast.Ident)
			// isidt = true
			fca.iscfn = true
		} else {
			selty := c.info.TypeOf(fca.selfn.X)
			fca.isrcver = !isinvalidty2(selty)
			fca.ispkgsel = ispackage(c.psctx, fca.selfn.X)

			selxty := c.info.TypeOf(fca.selfn.X)
			switch ne := selxty.(type) {
			case *types.Named:
				fca.isifacesel = isiface2(ne.Underlying())
			}
		}
		if idt, ok := fca.selfn.X.(*ast.Ident); ok {
			fca.isbuiltin = idt.Name == "builtin"
		}
	}
	gotyx := c.info.TypeOf(te.Fun)
	if gotyx == nil {
		log.Println(gotyx != nil, "wtfff", te.Fun, exprstr(te.Fun), exprpos(c.psctx, te))
	}
	if gotyx != nil {
		goty1, ok := gotyx.(*types.Signature)
		if ok {
			fca.fnty = goty1
			fca.isvardic = goty1.Variadic()
			fca.prmty = goty1.Params()
		} else {
			log.Println(gotyx, reflect.TypeOf(gotyx), te.Fun)
		}
	}

	// log.Println(te.Args, te.Fun, gotyx, reflect.TypeOf(gotyx), goty.Variadic())
	lexpr := c.psctx.kvpairs[te]
	fca.haslval = lexpr != nil
	if lexpr != nil {
		fca.lexpr = lexpr.(ast.Expr)
	}

	return fca
}
func (c *g2nc) genCallExprNorm(scope *ast.Scope, te *ast.CallExpr) {
	// funame := te.Fun.(*ast.Ident).Name
	fca := c.getCallExprAttr(scope, te)
	// gopp.Assert(!fca.isvardic, "moved", te.Fun)

	idt := newIdent(tmpvarname()) // store variadic args in array
	if fca.isvardic && fca.haslval {
		c.out("{0}").outfh().outnl()
	}
	if fca.isvardic {
		var elemsz interface{} = "sizeof(voidptr)"
		c.outf("cxarray2* %s = cxarray2_new(1, %v)", idt.Name, elemsz).outfh().outnl()
		prmty := fca.fnty.Params()
		elemty := prmty.At(prmty.Len() - 1).Type().(*types.Slice).Elem()
		// log.Println(te.Fun, prmty, reftyof(prmty), prmty.Len(), elemty, reftyof(elemty))
		for idx, e1 := range te.Args {
			if idx < fca.fnty.Params().Len()-1 {
				continue // non variadic arg
			}
			switch elty := elemty.(type) {
			case *types.Interface:
				tyname := c.exprTypeName(scope, e1)
				if tyname == "cxstring*" {
					tyname = "string"
				}
				tvar := tmpvarname()
				c.outf("voidptr %s= cxrt_type2eface((voidptr)&%s_metatype, (voidptr)&", tvar, tyname)
				c.genExpr(scope, e1)
				c.out(")").outfh().outnl()
				c.outf("cxarray2_append(%s, &%s)", idt.Name, tvar)
			default:
				_ = elty
				c.outf("cxarray2_append(%s, (voidptr)&", idt.Name)
				c.genExpr(scope, e1)
				c.out(")")
			}
			c.outfh().outnl()
		}
	}
	if fca.haslval {
		c.genExpr(scope, fca.lexpr)
		c.outeq()
	}

	if fca.isselfn {
		if fca.iscfn {
			c.out("(")
			c.genExpr(scope, fca.selfn.Sel)
			c.out(")")
		} else if fca.isifacesel {
			c.genExpr(scope, fca.selfn.X)
			c.out("->")
			c.genExpr(scope, fca.selfn.Sel)
		} else if fca.ispkgsel {
			c.genExpr(scope, fca.selfn.X)
			c.out(pkgsep)
			c.out(fca.selfn.Sel.Name)
		} else {
			// log.Println(selfn.X, reftyof(selfn.X), c.info.TypeOf(selfn.X))
			vartystr := c.exprTypeName(scope, fca.selfn.X)
			vartystr = strings.TrimRight(vartystr, "*")
			c.out(vartystr + "_" + fca.selfn.Sel.Name)
		}
	} else {
		fnobj := c.info.ObjectOf(te.Fun.(*ast.Ident))
		pkgo := fnobj.Pkg()
		if pkgo != nil && fnobj.Pkg() != nil {
			// c.out(fnobj.Pkg().Name(), "_")
		}
		c.genExpr(scope, te.Fun)
	}

	c.out("(")
	// reciever this
	if fca.isselfn && !fca.iscfn && !fca.ispkgsel && fca.isrcver {
		c.genExpr(scope, fca.selfn.X)
		c.out(gopp.IfElseStr(fca.isifacesel, "->data", ""))
		c.out(gopp.IfElseStr(len(te.Args) > 0, ",", ""))
	}

	for idx, e1 := range te.Args {
		if fca.isvardic && idx == fca.fnty.Params().Len()-1 {
			c.out(idt.Name)
			break
		}

		if fca.prmty != nil {
			prmn := fca.prmty.At(idx).Type()
			if _, ok := prmn.(*types.Interface); ok {
				c.out("cxrt_type2eface((voidptr)&")
				tyname := c.exprTypeName(scope, e1)
				if strings.Contains(tyname, "cxstring") {
					c.out("string")
				} else {
					c.out(tyname)
				}
				c.out("_metatype, (voidptr)&")
				c.genExpr(scope, e1)
				c.out(")")
			} else {
				c.genExpr(scope, e1)
			}
		} else {
			c.genExpr(scope, e1)
		}
		c.out(gopp.IfElseStr(idx == len(te.Args)-1, "", ", "))
	}
	c.out(")")
}
func (c *g2nc) genCallExprVaridic(scope *ast.Scope, te *ast.CallExpr) {
	// funame := te.Fun.(*ast.Ident).Name
	fca := c.getCallExprAttr(scope, te)
	gopp.Assert(fca.isvardic, "must", te.Fun)
	c.genCallExprNorm(scope, te)
}
func (c *g2nc) genTypeCtor(scope *ast.Scope, te *ast.CallExpr) {
	switch be := te.Fun.(type) {
	case *ast.Ident:
		switch be.Name {
		case "string":
			arg0 := te.Args[0]
			switch ce := arg0.(type) {
			case *ast.BasicLit:
				c.outf("cxstring_new_char(%v)", ce.Value)
			case *ast.Ident:
				varty := c.info.TypeOf(arg0)
				if isslicety2(varty) {
					c.outf("cxstring_new_cstr2((%v)->ptr, (%v)->len)", ce.Name, ce.Name)
				} else if iscstrty2(varty) {
					c.outf("cxstring_new_cstr(%v)", ce.Name)
				} else if funk.Contains(
					[]string{"voidptr", "charptr", "byteptr"}, varty.String()) {
					c.outf("cxstring_new_cstr(%v)", ce.Name)
				} else {
					c.outf("cxstring_new_char(%v) %v", ce.Name)
				}
			default:
				log.Println("todo", te.Fun, ce)
			}
		default:
			// log.Println("todo", te.Fun)
			c.outf("(%s)(", c.exprstr(te.Fun))
			c.genFuncArgs(scope, te.Args)
			c.outf(")")
		}
	case *ast.SelectorExpr:
		c.out("(")
		c.genExpr(scope, te.Fun)
		c.out(")")
		c.out("(")
		c.genFuncArgs(scope, te.Args)
		c.out(")")
	case *ast.ArrayType:
		c.out("cxstring_dup(")
		c.genExpr(scope, te.Args[0])
		c.out(")")
	default:
		log.Println("todo", te.Fun, be)
	}
}
func (c *g2nc) genFuncArgs(scope *ast.Scope, args []ast.Expr) {
	for idx, arg := range args {
		c.genExpr(scope, arg)
		if idx+1 < len(args) {
			c.out(",")
		}
	}
}
func (c *g2nc) genCallExprClosure(scope *ast.Scope, te *ast.CallExpr, fnlit *ast.FuncLit) {
	// funame := te.Fun.(*ast.Ident).Name
	lefte := c.valnames[te]
	selfn, isselfn := te.Fun.(*ast.SelectorExpr)
	_, isfnlit := te.Fun.(*ast.FuncLit)
	isidt := false
	iscfn := false
	ispkgsel := false
	if isselfn {
		var selidt *ast.Ident
		selidt, isidt = selfn.X.(*ast.Ident)
		iscfn = isidt && selidt.Name == "C"
		selty := c.info.TypeOf(selfn.X)
		ispkgsel = isinvalidty2(selty)
	}
	if !isselfn && isidt {
	}

	if lefte != nil {
		// {0} 只能用于初始化
		c.out("0").outfh().outnl()
	}

	closi := c.getclosinfo(fnlit)
	argtv := tmpvarname()
	c.out(closi.argtyname).outstar().outsp().out(argtv).outeq()
	c.outf("(%s*)cxmalloc(sizeof(%s))", closi.argtyname, closi.argtyname).outfh().outnl()
	for _, ido := range closi.idents {
		c.out(argtv, "->", ido.Name).outeq()
		c.out(ido.Name).outfh().outnl()
	}

	if lefte != nil {
		c.genExpr(scope, lefte)
		c.outeq()
	}

	if isselfn {
		if iscfn {
			c.genExpr(scope, selfn.Sel)
		} else {
			vartystr := c.exprTypeName(scope, selfn.X)
			vartystr = strings.TrimRight(vartystr, "*")
			c.out(vartystr + "_" + selfn.Sel.Name)
		}
	} else if isfnlit {
		closi := c.getclosinfo(fnlit)
		c.out(closi.fnname)
	} else {
		c.genExpr(scope, te.Fun)
	}

	c.out("(")
	if isselfn && !iscfn && !ispkgsel {
		c.genExpr(scope, selfn.X)
		c.out(gopp.IfElseStr(len(te.Args) > 0, ",", ""))
	}
	for idx, e1 := range te.Args {
		c.genExpr(scope, e1)
		c.out(gopp.IfElseStr(idx == len(te.Args)-1, "", ", "))
	}
	c.out(gopp.IfElseStr(len(te.Args) > 0, ",", ""))
	c.out(argtv)
	c.out(")")

	// check if real need, ;\n
	cs := c.psctx.cursors[te]
	if cs.Name() != "Args" {
		// c.outfh().outnl()
	}
}

// chan structure args
func (c *g2nc) genChanStargs(scope *ast.Scope, e ast.Expr) {
	var elemtyname = c.chanElemTypeName(e, false)
	var elemtyname2 = c.chanElemTypeName(e, true)
	// typedef struct { int  elem; } chan_arg_int;
	c.out("typedef struct {", elemtyname, " elem;} chan_arg_"+elemtyname2).outfh().outnl()
}
func (c *g2nc) genSendStmt(scope *ast.Scope, s *ast.SendStmt) {
	// var elemtyname = c.chanElemTypeName(s.Chan, false)
	var elemtyname2 = c.chanElemTypeName(s.Chan, true)
	var chanargname = "chan_arg_" + elemtyname2
	c.out("{").outnl()
	c.outf("%s* args = (%s*)cxmalloc(sizeof(%s))",
		chanargname, chanargname, chanargname).outfh().outnl()
	c.out("args->elem = ")
	c.genExpr(scope, s.Value)
	c.outfh().outnl()
	c.outf("cxrt_chan_send(")
	c.genExpr(scope, s.Chan)
	c.out(", args)").outfh().outnl()
	c.out("}").outnl()
}
func (c *g2nc) genRecvStmt(scope *ast.Scope, e ast.Expr) {
	var elemtyname = c.chanElemTypeName(e, false)
	var elemtyname2 = c.chanElemTypeName(e, true)
	var chanargname = "chan_arg_" + elemtyname2

	varobj := scope.Lookup("varname")
	if varobj != nil {
		c.outeq().out("{0}").outfh().outnl()
	}

	c.out("{")
	c.out("voidptr rvx = cxrt_chan_recv(")
	c.genExpr(scope, e)
	c.out(")").outfh().outnl()
	c.out(" // c = rv->v").outfh().outnl()
	c.outf("%s rvp = ((%s*)rvx)->elem", elemtyname, chanargname).outfh().outnl()

	if varobj != nil {
		c.genExpr(scope, varobj.Data.(ast.Expr)) // left
		c.out("= rvp").outfh().outnl()
	}

	c.out("}").outnl()
}
func (c *g2nc) chanElemTypeName(e ast.Expr, trimstar bool) string {
	var elemtyname = ""
	chtyx := c.info.TypeOf(e)
	switch t := chtyx.(type) {
	case *types.Chan:
		switch te := t.Elem().(type) {
		case *types.Basic:
			switch te.Kind() {
			case types.Int:
				elemtyname = "int"
			default:
				log.Println("unknown", te, te.Kind())
			}
		case *types.Pointer:
			tystr := c.exprTypeNameImpl2(nil, te, e)
			if trimstar {
				tystr = strings.Replace(tystr, "*", "p", -1)
			}
			return tystr
		default:
			log.Println("unknown", t, reflect.TypeOf(t.Elem()))
		}
	default:
		log.Println("unknown", chtyx)
	}
	if elemtyname == "" {
		log.Println("cannot resolve chan element typename", e, reflect.TypeOf(e))
	}
	return elemtyname
}
func (c *g2nc) genReturnStmt(scope *ast.Scope, e *ast.ReturnStmt) {
	fd := upfindFuncDeclNode(c.psctx, e, 0)
	ismret := fd.Type.Results.NumFields() >= 2

	if ismret {
		rtvname := c.multirets[fd]
		names := []*ast.Ident{}
		for _, fld := range fd.Type.Results.List {
			for _, name := range fld.Names {
				names = append(names, name)
			}
		}
		for idx, re := range e.Results {
			c.outf("%s->%s", rtvname.Name, tmpvarname2(idx))
			c.outeq()
			c.genExpr(scope, re)
			c.outfh().outnl()
		}
		if len(e.Results) == 0 {
			for idx, name := range names {
				c.outf("%s->%s", rtvname.Name, tmpvarname2(idx))
				c.outeq().out(name.Name)
				c.outfh().outnl()
			}
		}
		c.out("goto labmret").outfh().outnl()
	} else {
		c.genDeferStmt(scope, e)
		reses := []ast.Expr{}
		for idx, ae := range e.Results {
			if fd.Type.Results == nil {
				reses = append(reses, ae)
				continue
			}

			sigty := c.info.TypeOf(fd.Type.Results.List[idx].Type)
			resty := c.info.TypeOf(ae)
			reset := false

			switch ne := sigty.(type) {
			case *types.Named:
				if sigty != resty && isiface2(ne.Underlying()) {
					reset = true
					idt := newIdent(tmpvarname())
					reses = append(reses, idt)
					tystr := c.exprTypeName(scope, fd.Type.Results.List[idx].Type)
					c.out(tystr).outsp()
					c.genExpr(scope, idt)
					c.outeq()
					c.outf("%s_new_zero()", strings.Trim(tystr, "*")).outfh().outnl()
					undty := ne.Underlying().(*types.Interface)
					c.outf("%s->data =", idt.Name)
					c.genExpr(scope, ae)
					c.outfh().outnl()
					if isnilident(ae) {
						c.out(idt.Name).outeq().out("nilptr").outfh().outnl()
						break
					}
					for i := 0; i < undty.NumMethods(); i++ {
						c.outf("%s->%s = (__typeof__(%s->%s))%s_%s", idt.Name, undty.Method(i).Name(),
							idt.Name, undty.Method(i).Name(),
							strings.Trim(c.exprTypeName(scope, ae), "*"), undty.Method(i).Name())
						c.outfh().outnl()
					}
				}
			default:
				log.Println("todo", reflect.TypeOf(sigty))
			}
			if reset {
				// reses = append(reses, ae)
			} else {
				reses = append(reses, ae)
			}
		}
		if len(reses) < len(e.Results) {
			log.Println("todo", len(reses), len(e.Results), e.Results[0])
		}
		c.out("return").outsp()
		// log.Println(len(reses), len(e.Results), e.Results[0])
		for idx, _ := range e.Results {
			if idx >= len(reses) {
				break
			}

			c.genExpr(scope, reses[idx])
			c.out(gopp.IfElseStr(idx < len(e.Results)-1, ",", ""))
		}
	}
	// c.outfh().outnl().outnl()
}

// defer 也许可以用 goto label实现
func (c *g2nc) genDeferStmtSet(scope *ast.Scope, e *ast.DeferStmt) {
	deferi := c.getdeferinfo(e)
	tvname := tmpvarname()
	c.outf("int %s = %v", tvname, deferi.idx).outfh().outnl()
	c.outf("cxarray2_append(deferarr, (voidptr)&%s)", tvname)
}
func (c *g2nc) genDeferStmt(scope *ast.Scope, e ast.Stmt) {
	dstfd := upfindFuncDeclNode(c.psctx, e, 0)
	defers := []*ast.DeferStmt{}
	for _, defero := range c.psctx.defers {
		tmpfd := upfindFuncDeclNode(c.psctx, defero, 0)
		if tmpfd != dstfd {
			continue
		}
		defers = append(defers, defero)
	}
	// log.Println("got defered return", len(defers))
	c.outf("// defer section %v", len(defers)).outnl()
	if len(defers) == 0 {
		return
	}

	c.out("{").outnl()
	c.out("int deferarrsz = cxarray2_size(deferarr)").outfh().outnl()
	c.out("for (int deferarri = deferarrsz-1; deferarri>=0; deferarri--)")
	c.out("{").outnl()
	c.out("uintptr_t deferarrn = 0").outfh().outnl()
	c.out("*(uintptr_t*)cxarray2_get_at(deferarr, deferarri)").outfh().outnl()
	for i := 0; i < len(defers); i++ {
		defero := defers[i]
		c.out(gopp.IfElseStr(i > 0, "else", "")).outsp()
		c.outf("if (deferarrn == %d)", i)
		c.out("{").outnl()
		c.genExpr(scope, defero.Call)
		c.outfh().outnl()
		c.out("}").outnl()
	}
	c.out("}").outnl()
	c.out("}").outnl()
}

// keepvoid
// skiplast 作用于linebrk
func (this *g2nc) genFieldList(scope *ast.Scope, flds *ast.FieldList,
	keepvoid bool, withname bool, linebrk string, skiplast bool) {

	if keepvoid && (flds == nil || flds.NumFields() == 0) {
		this.out("void")
		return
	}
	if flds == nil {
		return
	}

	for idx, fld := range flds.List {
		_, _ = idx, fld
		log.Println(fld.Type, this.exprTypeName(scope, fld.Type))
		this.genTypeExpr(scope, fld.Type)
		this.outsp()
		if withname && len(fld.Names) > 0 {
			this.genExpr(scope, fld.Names[0])
		}
		outskip := skiplast && (idx == len(flds.List)-1)
		this.out(gopp.IfElseStr(outskip, "", linebrk))
	}
}

func (c *g2nc) genStructZeroFields(scope *ast.Scope) {
	log.Println("zero struct fields")
}

func (this *g2nc) genTypeExpr(scope *ast.Scope, e ast.Expr) {
	this.out(this.exprTypeName(scope, e))
}

func (c *g2nc) genExpr(scope *ast.Scope, e ast.Expr) {
	varname := scope.Lookup("varname")
	if varname != nil {
		vartyp := c.info.TypeOf(varname.Data.(ast.Expr))
		log.Println(vartyp, varname)
		if iseface2(vartyp) {
			_, iscallexpr := e.(*ast.CallExpr)
			_, isidt := e.(*ast.Ident)
			// _, lisidt := varname.Data.(ast.Expr).(*ast.Ident)
			if !iscallexpr && !isidt {
				// vartyp2 := reflect.TypeOf(varname.Data.(ast.Expr))
				c.out("(cxeface*){0}").outfh().outnl()

				tmpvar := tmpvarname()
				c.out(c.exprTypeName(scope, e), tmpvar, "=")
				ns := putscope(scope, ast.Var, "varname", newIdent(tmpvar))
				c.genExpr2(ns, e)
				c.outfh().outnl()
				c.genExpr2(scope, varname.Data.(ast.Expr))
				c.outeq()
				ety := c.info.TypeOf(e)
				switch ety.(type) {
				case *types.Interface:
					c.out(tmpvar)
				default: // convert
					c.outf("cxeface_new_of2((voidptr)&%s, sizeof(%s))", tmpvar, tmpvar)
				}
				return
			}
		}
	}
	c.genExpr2(scope, e)
}
func (this *g2nc) genExpr2(scope *ast.Scope, e ast.Expr) {
	// log.Println(reflect.TypeOf(e), e)
	switch te := e.(type) {
	case *ast.Ident:
		idname := te.Name
		idname = gopp.IfElseStr(idname == "nil", "nilptr", idname)
		idname = gopp.IfElseStr(idname == "string", "cxstring*", idname)
		if strings.HasPrefix(idname, "_Ctype_") {
			idname = idname[7:]
		}
		eobj := this.info.ObjectOf(te)
		log.Println(e, eobj, isglobalid(this.psctx, te))
		if eobj != nil {
			pkgo := eobj.Pkg()
			if pkgo != nil {
				// this.out(pkgo.Name())
			}
		}
		// TODO 要查看是否有上级,否则无法判断包前缀
		if strings.HasPrefix(idname, "_Cfunc_") || isglobalid(this.psctx, te) {
			eobj := this.info.ObjectOf(te)
			if eobj != nil && eobj.Pkg().Name() == "C" {
			} else {
				this.out(this.pkgpfx())
			}
		}
		this.out(idname, "")
	case *ast.ArrayType:
		tystr := this.exprstr(te)
		if tystr == "[0]byte" {
			this.out("void")
			break
		}
		log.Println("todo", te, reflect.TypeOf(e), e.Pos())
		this.out(tystr)
	case *ast.StructType:
		this.genFieldList(scope, te.Fields, false, true, ";\n", false)
	case *ast.UnaryExpr:
		// log.Println(te.Op.String(), te.X)
		switch te.Op {
		case token.ARROW:
			this.genRecvStmt(scope, te.X)
			return
		default:
			// log.Println("unknown", te.Op.String())
		}
		keepop := true
		switch t2 := te.X.(type) {
		case *ast.CompositeLit:
			if iscsel(t2.Type) {
				ste := t2.Type.(*ast.SelectorExpr)
				// this.outf("// c struct ctor %s", ste.Sel.Name)
				this.outf("cxmalloc(sizeof(%s))", ste.Sel.Name)
				keepop = false
				break
			}

			tystr := this.exprTypeName(scope, t2.Type)
			this.outf("%s_new_zero()", tystr) //.outnl()
			this.outfh().outnl()
			keepop = false
			varname := scope.Lookup("varname")

			tyobj := this.info.ObjectOf(t2.Type.(*ast.Ident))
			goty := tyobj.Type().(*types.Named).Underlying().(*types.Struct)
			for idx, elmx := range t2.Elts {
				// log.Println(elmx, goty, goty.Field(idx), reflect.TypeOf(elmx))
				switch elme := elmx.(type) {
				case *ast.KeyValueExpr:
					this.outf("%s->%s", varname.Data, elme.Key)
					this.outeq()
					this.genExpr(scope, elme.Value)
					this.outfh().outnl()
				default:
					fld := goty.Field(idx)
					this.outf("%s->%s", varname.Data, fld.Name())
					this.outeq()
					this.genExpr(scope, elmx)
					this.outfh().outnl()
				}
			}
		case *ast.UnaryExpr:
			log.Println(t2, t2.X, t2.Op)
		default:
			log.Println(reflect.TypeOf(te), t2, reflect.TypeOf(te.X), te.Pos())
		}
		if keepop {
			this.outf("%v", te.Op.String())
			this.genExpr(scope, te.X)
		}
	case *ast.CompositeLit:
		switch be := te.Type.(type) {
		case *ast.MapType:
			this.outf("cxhashtable_new()").outfh().outnl()
			var vo = scope.Lookup("varname")
			for idx, ex := range te.Elts {
				switch be := ex.(type) {
				case *ast.KeyValueExpr:
					this.genCxmapAddkv(scope, vo.Data, be.Key, be.Value)
					this.outfh().outnl()
				default:
					log.Println("unknown", idx, reflect.TypeOf(ex))
				}
			}
		case *ast.ArrayType:
			var vo = scope.Lookup("varname")
			if vo == nil {
				gotyval := this.info.Types[te]
				log.Println("temp var?", vo, this.exprpos(te), gotyval)
			}
			bety := this.info.TypeOf(be.Elt)
			var elemsz interface{} = uintptr((&types.StdSizes{}).Sizeof(bety))
			elemsz = gopp.IfElse(elemsz == uintptr(0), "sizeof(voidptr)", elemsz)
			gopp.Assert(elemsz != 0, "wtfff", elemsz, bety)
			this.outf("cxarray2_new(1, %v)", elemsz).outfh().outnl()
			for idx, ex := range te.Elts {
				log.Println(vo == nil, ex, idx, this.exprpos(ex))
				this.genCxarrAdd(scope, vo.Data, ex, idx)
				this.outfh().outnl()
			}
			if be == nil {
			}
		case *ast.Ident: // TODO
			var vo = scope.Lookup("varname")
			this.outf("%v_new_zero()", this.exprTypeName(scope, be)).outfh().outnl()
			for _, ex := range te.Elts {
				this.outf("%v->%v = %v", vo, "aaa", ex)
				this.outfh().outnl()
			}
		default:
			log.Println("todo", te.Type, reflect.TypeOf(te.Type))
		}

	case *ast.CallExpr:
		if idto, ok := te.Fun.(*ast.Ident); ok {
			funame := idto.Name
			if funk.Contains([]string{"_cgoCheckPointer"}, funame) {
				this.out("//").outsp()
			}
		}
		this.genCallExpr(scope, te)
	case *ast.BasicLit:
		ety := this.info.TypeOf(e)
		if ety == nil { // we created
			this.out(te.Value)
			break
		}
		switch t := ety.Underlying().(type) {
		case *types.Basic:
			switch t.Kind() {
			case types.Int, types.UntypedInt, types.UntypedRune,
				types.Uint, types.Int64, types.Uint64:
				this.out(te.Value)
			case types.String, types.UntypedString:
				this.out(fmt.Sprintf("cxstring_new_cstr(%s)", te.Value))
			case types.Float64, types.Float32, types.UntypedFloat:
				this.out(te.Value)
			case types.Uint8, types.Int8, types.Uint32, types.Int32:
				this.out(te.Value)
			case types.Uintptr:
				this.out(te.Value)
			default:
				if isctydeftype2(t) {
					this.out(te.Value)
				} else {
					log.Println("unknown", t.String())
				}
			}
		default:
			log.Println("unknown", t, reflect.TypeOf(t))
		}
	case *ast.BinaryExpr:
		opty := this.info.TypeOf(te.X)
		if isstrty2(opty) {
			switch te.Op {
			case token.EQL:
				this.out("cxstring_eq(")
			case token.NEQ:
				this.out("cxstring_ne(")
			case token.ADD:
				this.out("cxstring_add(")
			default:
				log.Println("todo", te.Op)
			}
			this.genExpr(scope, te.X)
			this.out(",")
			this.genExpr(scope, te.Y)
			this.out(")")
		} else {
			this.genExpr(scope, te.X)
			this.out(te.Op.String())
			this.genExpr(scope, te.Y)
		}
	case *ast.ChanType:
		this.out("voidptr")
	case *ast.IndexExpr:
		varty := this.info.TypeOf(te.X)
		vo := scope.Lookup("varval")
		if varty == nil { // c type ???
			// gopp.Assert(1 == 2, "waitdep", te)
			this.genExpr(scope, te.X)
			this.out("[")
			this.genExpr(scope, te.Index)
			this.out("]")
		} else if isctydeftype2(varty) {
			this.genExpr(scope, te.X)
			this.out("[")
			this.genExpr(scope, te.Index)
			this.out("]")
		} else if ismapty(varty.String()) {
			if vo == nil {
				this.genCxmapAddkv(scope, te.X, te.Index, nil)
			} else {
				this.genCxmapAddkv(scope, te.X, te.Index, vo.Data)
			}
		} else if isslicety(varty.String()) {
			// get or set?
			if vo == nil { // right value
				this.genCxarrGet(scope, te.X, te.Index, varty)
			} else { // left value
				this.genCxarrSet(scope, te.X, te.Index, vo.Data)
			}
		} else if isstrty(varty.String()) {
			if vo == nil { // right value
				this.out("((cxstring*)")
				this.genExpr(scope, te.X)
				this.out(")->ptr[")
				this.genExpr(scope, te.Index)
				this.out("]")
			} else { // left value
				this.out("((cxstring*)")
				this.genExpr(scope.Outer, te.X) // temporarily left value
				this.out(")->ptr[")
				this.genExpr(scope, te.Index)
				this.out("]")
				this.out("=")
				this.genExpr(scope, vo.Data.(ast.Expr))
			}
		} else if isinvalidty2(varty) { // index of c type???
			gopp.Assert(1 == 2, "waitdep", varty)
			this.genExpr(scope, te.X)
			this.out("[")
			this.genExpr(scope, te.Index)
			this.out("]")
		} else if varty.String() == "byte" { // multiple dimission index of c type???
			this.genExpr(scope, te.X)
			this.out("[")
			this.genExpr(scope, te.Index)
			this.out("]")
		} else {
			log.Println("todo", te.X, te.Index, exprstr(te))
			log.Println("todo", reftyof(te.X), varty.String(), this.exprpos(te.X))
		}
	case *ast.SliceExpr:
		varty := this.info.TypeOf(te.X)
		lowe := te.Low
		highe := te.High
		if lowe == nil {
			lowe = newLitInt(0)
		}
		if isstrty2(varty) {
			this.outf("cxstring_sub(%v, ", te.X)
			this.genExpr(scope, lowe)
			this.out(",")

			if highe == nil {
				this.outf("(%v)->len", te.X)
			} else {
				this.genExpr(scope, te.High)
			}
			this.out(")")
		} else if isslicety2(varty) {
			this.outf("cxarray2_slice(%v, ", te.X)
			this.genExpr(scope, lowe)
			this.out(",")

			if highe == nil {
				this.outf("cxarray2_size(%v)", te.X)
			} else {
				this.genExpr(scope, te.High)
			}
			this.out(")")
		} else {
			log.Println("todo", varty, te)
		}
	case *ast.SelectorExpr:
		if iscsel(te) {
		} else {
			this.genExpr(scope, te.X)
			selxty := this.info.TypeOf(te.X)
			log.Println(selxty, reflect.TypeOf(selxty), te.X)
			if selxty == nil {
				gopp.Assert(1 == 2, "waitdep", te)
				// c type?
				this.out(". /* c struct selctorexpr */")
			} else if isinvalidty2(selxty) { // package
				this.out("_")
			} else {
				switch selxty.(type) {
				case *types.Named:
					this.out(".")
				case *types.Pointer:
					this.out("->")
				default:
					if isctydeftype2(selxty) {
						this.out(".")
					} else {
						this.out("->")
					}
				}
			}
		}
		this.genExpr(scope, te.Sel)
		this.outsp()
	case *ast.StarExpr:
		idt, isidt := te.X.(*ast.Ident)
		if isidt {
			varobj := this.psctx.info.ObjectOf(idt)
			if istypety(varobj.String()) {
				this.genExpr(scope, te.X)
				this.out("*")
			} else if isvarty(varobj.String()) {
				this.out("*")
				this.genExpr(scope, te.X)
			} else {
				log.Println("todo", varobj.Type(), varobj.String())
			}
		} else {
			this.out("*")
			this.genExpr(scope, te.X)
		}
	case *ast.InterfaceType:
		if te.Methods != nil && te.Methods.NumFields() > 0 {
			this.out("cxiface")
		} else {
			this.out("cxeface")
		}
	case *ast.TypeAssertExpr:
		tystr := this.exprTypeName(scope, te.Type)
		this.outf("(%s)(", tystr)
		this.genExpr(scope, te.X)
		this.out("->data)")
	case *ast.ParenExpr:
		this.out("(")
		this.genExpr(scope, te.X)
		this.out(")")
	case *ast.FuncLit:
		closi := this.getclosinfo(te)
		this.out(closi.fnname).outfh().outnl()
	default:
		log.Println("unknown", reflect.TypeOf(e), e, te)
	}
}
func (c *g2nc) genCxmapAddkv(scope *ast.Scope, vnamex interface{}, ke ast.Expr, vei interface{}) {
	keystr := ""
	switch be := ke.(type) {
	case *ast.BasicLit:
		switch be.Kind {
		case token.STRING:
			keystr = fmt.Sprintf("cxstring_new_cstr(%s)", be.Value)
		default:
			// log.Println("unknown index key kind", be.Kind)
			keystr = fmt.Sprintf("%v", be.Value)
		}
	case *ast.Ident:
		varty := c.info.TypeOf(ke)
		switch varty.String() {
		case "string":
			keystr = be.Name
		default:
			log.Println("unknown", varty, ke)
		}
	case *ast.SelectorExpr:
		varty := c.info.TypeOf(ke)
		switch varty.String() {
		case "string":
			sym := fmt.Sprintf("%v->%v", be.X, be.Sel)
			keystr = sym
		default:
			log.Println("unknown", varty, ke)
		}
	default:
		log.Println("unknown index key", ke, reflect.TypeOf(ke))
	}

	valstr := ""
	switch be := vei.(type) {
	case *ast.BasicLit:
		valstr = be.Value
	case *ast.Ident:
		valstr = be.Name
	default:
		log.Println("unknown", vei, reflect.TypeOf(ke), reflect.TypeOf(vei))
	}

	varstr := fmt.Sprintf("%v", vnamex)
	switch be := vnamex.(type) {
	case *ast.Ident:
	case *ast.SelectorExpr:
		varstr = fmt.Sprintf("%v->%v", be.X, be.Sel)
	default:
		log.Println(vnamex, reflect.TypeOf(vnamex))
	}

	c.outf("hashtable_add(%v, (voidptr)(uintptr_t)%v, (voidptr)(uintptr_t)%s)",
		varstr, keystr, valstr) // .outfh().outnl()
}
func (c *g2nc) genCxarrAdd(scope *ast.Scope, vnamex interface{}, ve ast.Expr, idx int) {
	// log.Println(vnamex, ve, idx)
	valstr := ""
	switch be := ve.(type) {
	case *ast.BasicLit:
		switch be.Kind {
		case token.STRING:
			valstr = fmt.Sprintf("cxstring_new_cstr(%v)", be.Value)
		case token.INT:
			valstr = be.Value
		default:
			log.Println("todo", ve, idx, reflect.TypeOf(ve))
		}
	default:
		log.Println("unknown", ve, reflect.TypeOf(ve))
	}

	varobj := c.info.ObjectOf(vnamex.(*ast.Ident))
	pkgpfx := gopp.IfElseStr(isglobalid(c.psctx, vnamex.(*ast.Ident)), varobj.Pkg().Name(), "")
	pkgpfx = gopp.IfElseStr(pkgpfx == "", "", pkgpfx+"_")
	c.outf("cxarray2_append(%s%v, (voidptr)&%v)", pkgpfx,
		vnamex.(*ast.Ident).Name, valstr) // .outfh().outnl()
}
func (c *g2nc) genCxarrSet(scope *ast.Scope, vname ast.Expr, vidx ast.Expr, elem interface{}) {
	idxstr := ""
	valstr := ""

	switch te := vidx.(type) {
	case *ast.BasicLit:
		idxstr = te.Value
	default:
		log.Println("todo", vidx, reflect.TypeOf(vidx))
	}

	switch te := elem.(type) {
	case *ast.BasicLit:
		valstr = te.Value
	default:
		log.Println("todo", elem, reflect.TypeOf(elem))
	}

	c.outf("cxarray2_replace_at(%v, (voidptr)(uintptr_t)%v, %v, nilptr)",
		vname, valstr, idxstr).outfh().outnl()
}
func (c *g2nc) genCxarrGet(scope *ast.Scope, vname ast.Expr, vidx ast.Expr, varty types.Type) {
	var elemty types.Type
	switch arrty := varty.(type) {
	case *types.Slice:
		elemty = arrty.Elem()
	case *types.Array:
		elemty = arrty.Elem()
	}
	tystr := c.exprTypeName(scope, vname)
	tystr = c.exprTypeNameImpl2(scope, elemty, nil)
	c.outf("*(%v*)", tystr)
	c.outf("cxarray2_get_at(")
	c.genExpr(scope, vname)
	c.out(",")
	c.genExpr(scope, vidx)
	c.out(")").outnl()
}
func (this *g2nc) exprTypeName(scope *ast.Scope, e ast.Expr) string {
	// log.Println(e, reflect.TypeOf(e))
	tyname := this.exprTypeNameImpl(scope, e)
	// log.Println(exprstr(e), reftyof(e), tyname)
	if tyname == "unknownty" {
		// log.Panicln(tyname, e, reflect.TypeOf(e), this.exprpos(e))
	}
	return tyname
}
func (this *g2nc) exprTypeNameImpl(scope *ast.Scope, e ast.Expr) string {

	{
		// return "unknownty"
	}
	{ // C.xxx or C.xxx()
		if iscsel(e) {
			name := exprstr(e)[2:]
			return fmt.Sprintf("%s__ctype", name)
		}
		if ce, ok := e.(*ast.CallExpr); ok {
			if iscsel(ce.Fun) {
				name := exprstr(ce.Fun)[2:]
				return fmt.Sprintf("%s__ctype", name)
			}
			log.Println(ce.Fun, reftyof(ce.Fun), len(this.info.Types))
			if idt, ok := ce.Fun.(*ast.Ident); ok {
				if funk.Contains([]string{"int"}, idt.Name) {
					return idt.Name
				}
			}
		}
		if se, ok := e.(*ast.StarExpr); ok {
			log.Println(se, reftyof(se), se.X, reftyof(se.X))
			if iscsel(se.X) {
				name := exprstr(se.X)[2:]
				return fmt.Sprintf("%s__ctype", name)
			}
			if ce, ok := se.X.(*ast.CallExpr); ok {
				if iscsel(ce.Fun) {
					name := exprstr(ce.Fun)[2:]
					return fmt.Sprintf("%s__ctype", name)
				}
			}
		}
		if se, ok := e.(*ast.UnaryExpr); ok {
			log.Println(se, reftyof(se), se.X, reftyof(se.X))
			if iscsel(se.X) {
				name := exprstr(se.X)[2:]
				return fmt.Sprintf("%s__ctype", name)
			}
			if ce, ok := se.X.(*ast.CompositeLit); ok {
				if iscsel(ce.Type) {
					name := exprstr(ce.Type)[2:]
					return fmt.Sprintf("%s__ctype*", name)
				}
			}
		}
		if ie, ok := e.(*ast.IndexExpr); ok {
			log.Println(exprstr(ie), ie.X, reftyof(ie.X), this.info.TypeOf(ie.X))
			xty := this.info.TypeOf(ie.X)
			// gopp.Assert(xty != nil, "waitdep", ie)
			if (xty == nil || isinvalidty2(xty)) || isctydeftype2(xty) {
				// c type???
				dimn := strings.Count(exprstr(ie), "[")
				dimstr := strings.Repeat("[0]", dimn)
				tope := ie.X
				for i := 0; i < dimn-1; i++ {
					ie2 := ie.X.(*ast.IndexExpr)
					tope = ie2.X
				}
				return fmt.Sprintf("__typeof__(%s%s)", tope, dimstr)
			} else if xty != nil && xty.String() == "byte" {
				// multiple dimision index of c type???
				dimn := strings.Count(exprstr(ie), "[")
				dimstr := strings.Repeat("[0]", dimn)
				tope := ie.X
				for i := 0; i < dimn-1; i++ {
					ie2 := ie.X.(*ast.IndexExpr)
					tope = ie2.X
				}
				return fmt.Sprintf("__typeof__(%s%s)", tope, dimstr)
			}
		}
	}

	goty := this.info.TypeOf(e)
	if goty == nil {
		if ie, ok := e.(*ast.CallExpr); ok {
			log.Println(ie.Fun, reftyof(ie.Fun), this.info.TypeOf(ie.Fun))
			if _, ok2 := ie.Fun.(*ast.ParenExpr); ok2 {
				goty = this.info.TypeOf(ie.Fun)
			}
		}
	}
	if goty == nil {
		log.Println(e, exprstr(e), reftyof(e), this.exprpos(e))
		if exprstr(e) == "(bad expr)" {
			return "int"
		}
		if exprstr(e) == mthsep {
			return "int"
		}
		log.Panicln(e, exprstr(e), reftyof(e), this.exprpos(e))
	}
	val := this.exprTypeNameImpl2(scope, goty, e)
	if isinvalidty(val) {
		// log.Panicln("unreachable")
		val = this.exprstr(e)
		val = strings.Replace(val, "C.", "", 1)
		log.Println(val, exprstr(e))
		// log.Panicln(e, iscsel(e), this.exprpos(e), this.exprstr(e), sign2rety(val))
		return sign2rety(val)
	}
	return val
}
func (this *g2nc) exprTypeNameImpl2(scope *ast.Scope, ety types.Type, e ast.Expr) string {

	{
		// return "unknownty"
	}

	goty := ety
	tyval, isudty := this.strtypes[goty.String()]
	log.Println(goty, reftyof(goty), e, reftyof(e), exprstr(e))

	switch te := goty.(type) {
	case *types.Basic:
		if isstrty(te.Name()) {
			return "cxstring*"
		} else {
			if strings.Contains(te.Name(), "string") {
				log.Println(te.Name())
			}
			// log.Println(te, reftyof(e), te.Info(), te.Name(), te.Underlying(), reftyof(te.Underlying()))
			tystr := strings.Replace(te.String(), ".", "_", 1)
			if strings.HasPrefix(tystr, "untyped ") {
				tystr = tystr[8:]
			}
			return tystr
			// return te.Name()
		}
	case *types.Named:
		teobj := te.Obj()
		pkgo := teobj.Pkg()
		undty := te.Underlying()
		log.Println(teobj, pkgo, undty)
		switch ne := undty.(type) {
		case *types.Interface:
			if pkgo == nil { // builtin???
				return teobj.Name() + "*"
			}
			return fmt.Sprintf("%s%s%s", pkgo.Name(), pkgsep, teobj.Name())
		case *types.Struct:
			tyname := teobj.Name()
			if strings.HasPrefix(tyname, "_Ctype_") {
				return fmt.Sprintf("%s%s%s", pkgo.Name(), pkgsep, tyname[7:])
			}
			if pkgo.Name() == "C" {
				return fmt.Sprintf("%s", tyname)
			}
			return fmt.Sprintf("%s%s%s", pkgo.Name(), pkgsep, teobj.Name())
		case *types.Basic:
			tyname := teobj.Name()
			if strings.HasPrefix(tyname, "_Ctype_") {
				return tyname[7:]
			}
			return fmt.Sprintf("%s%s%s", pkgo.Name(), pkgsep, teobj.Name())
		case *types.Array:
			tyname := teobj.Name()
			if strings.HasPrefix(tyname, "_Ctype_") {
				return tyname[7:]
			}
		default:
			gopp.G_USED(ne)
		}
		log.Println("todo", teobj.Name(), reflect.TypeOf(undty), goty)
		return "/*todo*/" + teobj.Name()
		// return sign2rety(te.String())
	case *types.Pointer:
		tystr := this.exprTypeNameImpl2(scope, te.Elem(), e)
		tystr += "*"
		// log.Println(tystr, reftyof(te.Elem()))
		return tystr
	case *types.Slice:
		tystr := te.String()
		if tystr == "[0]byte" {
			return "void"
		}
		return "cxarray2*"
	case *types.Array:
		tystr := te.String()
		if tystr == "[0]byte" {
			return "void"
		}
		return "cxarray2*"
	case *types.Chan:
		return "voidptr"
	case *types.Map:
		return "HashTable*"
	case *types.Signature:
		switch fe := e.(type) {
		case *ast.FuncLit:
			if closi, ok := this.closidx[fe]; ok {
				return closi.fntype
			} else {
				log.Println("todo", goty, reflect.TypeOf(goty), isudty, tyval, te)
			}
		case *ast.Ident:
			return te.String()
		}
		return te.String()
	case *types.Interface:
		return "cxeface*"
	case *types.Tuple:
		log.Println(e, reflect.TypeOf(e))
		switch ce := e.(type) {
		case *ast.CallExpr:
			return fmt.Sprintf("%v_multiret_arg*", ce.Fun)
		case *ast.TypeAssertExpr:
			return fmt.Sprintf("%v_multiret_arg*", "aaa")
		default:
			log.Println("todo", goty, reflect.TypeOf(goty), isudty, tyval, te, this.exprpos(e))
		}
	default:
		log.Println("todo", goty, exprstr(e), reftyof(goty), isudty, tyval, te, this.exprpos(e))
		return te.String() + "/*todo*/"
	}

	panic("unreachable")
}
func (this *g2nc) exprTypeFmt(scope *ast.Scope, e ast.Expr) string {
	goty := this.info.TypeOf(e)
	if goty == nil {
		// maybe not exist func? like c function?
		return "d-nilty"
	}
	tyval, isudty := this.strtypes[goty.String()]
	// log.Println(goty, reflect.TypeOf(goty), tyval, isudty, e, reflect.TypeOf(e))

	switch te := goty.(type) {
	case *types.Basic:
		if isstrty(te.Name()) {
			return ".*s"
		} else {
			switch te.Kind() {
			case types.Float32, types.Float64:
				return "g" // wow
				// return "f"
			case types.Byteptr:
				return "s"
			case types.Voidptr:
				return "p"
			default:
				log.Println(exprstr(e), te, te.Kind(), goty)
				return "d"
			}
		}
	case *types.Named:
		segs := strings.Split(te.String(), "._Ctype_")
		if len(segs) == 2 {
			switch segs[1] {
			case "int", "int32", "int64", "long", "short":
				return "d"
			case "float", "double":
				return "g"
				// return "f"
			default:
				log.Println("todo", segs)
			}
		}
		return "p"
	case *types.Pointer:
		return "p"
	case *types.Slice, *types.Array:
		return "p"
	case *types.Map:
		return "p"
	default:
		log.Println(goty, reflect.TypeOf(goty), isudty, tyval, te)
		return "d-wt"
	}

	panic("unreachable")
}

func (this *g2nc) genGenDecl(scope *ast.Scope, d *ast.GenDecl) {
	// log.Println(d.Tok, d.Specs, len(d.Specs), d.Tok.IsKeyword(), d.Tok.IsLiteral(), d.Tok.IsOperator())
	for idx, spec := range d.Specs {
		switch tspec := spec.(type) {
		case *ast.TypeSpec:
			this.genTypeSpec(scope, tspec)
		case *ast.ValueSpec:
			this.genValueSpec(scope, tspec, idx)
		case *ast.ImportSpec:
			// log.Println("todo", reflect.TypeOf(d), reflect.TypeOf(spec), tspec.Path, tspec.Name)
			this.outf("// import %v by %s", tspec.Path, this.exprpos(tspec)).outnl().outnl()
			// log.Println(tspec.Comment)
		default:
			log.Println("unknown", reflect.TypeOf(d), reflect.TypeOf(spec))
		}
	}
}
func (this *g2nc) genTypeSpec(scope *ast.Scope, spec *ast.TypeSpec) {
	log.Println(spec.Type, reflect.TypeOf(spec.Type), spec.Name)
	this.outf("// %s", this.exprpos(spec).String()).outnl()
	switch te := spec.Type.(type) {
	case *ast.StructType:
		specname := trimCtype(spec.Name.Name)
		this.outf("typedef struct %s%s %s%s",
			this.pkgpfx(), specname, this.pkgpfx(), specname).outfh().outnl()
		this.outf("struct %s%s {", this.pkgpfx(), specname)
		this.outnl()
		this.genFieldList(scope, te.Fields, false, true, ";", false)
		this.out("}").outfh().outnl()
		this.outnl()
		this.outf("static const _metatype %s%s_metatype = {", this.pkgpfx(), specname)
		this.outnl()
		this.outf(".kind = %d,", reflect.Struct).outnl()
		this.outf(".size = sizeof(%s%s),", this.pkgpfx(), specname).outnl()
		this.outf(".size = alignof(%s%s),", this.pkgpfx(), specname).outnl()
		this.outf(".tystr = \"%s%s\"", this.pkgpfx(), specname).outnl()
		this.out("}").outfh().outnl()
		this.outnl()
		this.out("static").outsp()
		this.outf("%s%s* %s%s_new_zero() {",
			this.pkgpfx(), specname, this.pkgpfx(), specname).outnl()
		this.outf("  %s%s* obj = (%s%s*)cxmalloc(sizeof(%s%s))",
			this.pkgpfx(), specname, this.pkgpfx(), specname,
			this.pkgpfx(), specname).outfh().outnl()
		for _, fld := range te.Fields.List {
			fldty := this.info.TypeOf(fld.Type)
			for _, fldname := range fld.Names {
				if isstrty2(fldty) {
					this.outf("obj->%s = cxstring_new()", fldname.Name).outfh().outnl()
				} else if isslicety2(fldty) {
					var elemsz interface{} = unsafe.Sizeof(uintptr(0))
					elemsz = gopp.IfElse(elemsz == uintptr(0), "sizeof(voidptr)", elemsz)
					this.outf("obj->%s = cxarray2_new(1, %v)", fldname.Name, elemsz).outfh().outnl()
				} else if ismapty2(fldty) {
					this.outf("obj->%s = cxhashtable_new()", fldname.Name).outfh().outnl()
				} else if ischanty2(fldty) {
					log.Println("how to", fld.Type.(*ast.ChanType).Value)
					this.outf("obj->%s = cxrt_chan_new(0)", fldname.Name).outfh().outnl()
				}
			}
		}
		this.out("  return obj").outfh().outnl()
		this.out("}").outnl()
		this.outnl()
	case *ast.Ident:
		log.Println(spec.Type, reflect.TypeOf(spec.Type), te)
		tystr := this.exprTypeName(scope, spec.Type)
		specname := trimCtype(spec.Name.Name)
		this.outf("typedef %v %s%v", tystr, this.pkgpfx(), specname).outfh().outnl()
		// this.outf("typedef %v %s%v", spec.Type, this.pkgpfx(), spec.Name.Name).outfh().outnl()
	case *ast.StarExpr:
		log.Println(spec.Type, reflect.TypeOf(spec.Type), te.X, reflect.TypeOf(te.X), spec.Name)
		this.out("typedef").outsp()
		// this.out(this.pkgpfx())
		// this.genExpr(scope, te.X)
		this.out(this.exprTypeName(scope, te.X))
		this.out("*").outsp()
		specname := trimCtype(spec.Name.Name)
		this.out(this.pkgpfx() + specname)
		this.outfh().outnl()
	case *ast.SelectorExpr:
		this.out("typedef").outsp()
		this.genExpr(scope, te.X)
		this.out("_")
		this.genExpr(scope, te.Sel)
		this.outsp()
		specname := trimCtype(spec.Name.Name)
		this.out(this.pkgpfx() + specname)
		this.outfh().outnl()
	case *ast.InterfaceType:
		this.outf("typedef struct %s%s %s%s", this.pkgpfx(), spec.Name,
			this.pkgpfx(), spec.Name).outfh().outnl()
		this.outf("struct %s%s {", this.pkgpfx(), spec.Name).outnl()
		this.out("voidptr value").outfh().outnl()
		for _, fld := range te.Methods.List {
			switch fldty := fld.Type.(type) {
			case *ast.FuncType:
				for _, name := range fld.Names {
					this.genFieldList(scope, fldty.Results, true, false, "", true)
					this.outf("(*%s)(voidptr", name.Name)
					this.out(gopp.IfElseStr(fldty.Params.NumFields() > 0, ",", ""))
					this.genFieldList(scope, fldty.Params, false, false, "", true)
					this.out(")").outfh().outnl()
				}
			default:
				log.Println("todo", fld.Type, reflect.TypeOf(fld.Type))
			}
		}
		this.out("}").outfh().outnl()
		this.outf("%s%s* %s%s_new_zero() {", this.pkgpfx(), spec.Name.Name, this.pkgpfx(), spec.Name.Name)
		this.outf("return").outsp()
		this.outf("(%s%s*)cxmalloc(sizeof(%s%s))", this.pkgpfx(), spec.Name.Name, this.pkgpfx(), spec.Name.Name)
		this.outfh().outnl()
		this.out("}").outnl().outnl()
	case *ast.ArrayType:
		log.Println("todo", spec.Name, spec.Type, reflect.TypeOf(spec.Type), te)
		log.Println("todo", te.Elt, te.Len, this.exprstr(te))
		tystr := this.exprstr(te)
		if tystr == "[0]byte" {
			log.Println("todo", "hehe")
			// this.out("void...")
			break
		}
		this.out("todo", tystr)
	default:
		log.Println("todo", spec.Name, spec.Type, reflect.TypeOf(spec.Type), te)
	}
}
func putscope(scope *ast.Scope, k ast.ObjKind, name string, value interface{}) *ast.Scope {
	var pscope = ast.NewScope(scope)
	var varobj = ast.NewObj(k, name)
	varobj.Data = value
	pscope.Insert(varobj)
	return pscope
}

// TODO depcreate
var vp1stval ast.Expr
var vp1stty types.Type
var vp1stidx int

func (c *g2nc) genValueSpec(scope *ast.Scope, spec *ast.ValueSpec, validx int) {
	cs := c.psctx.cursors[spec]
	pcs := cs.Parent()
	isconst := false
	if d, ok := pcs.(*ast.GenDecl); ok {
		isconst = d.Tok == token.CONST
	}
	isglobvar := c.psctx.isglobal(spec)

	for idx, varname := range spec.Names {
		varty := c.info.TypeOf(spec.Type)
		if varty == nil && idx < len(spec.Values) {
			varty = c.info.TypeOf(spec.Values[idx])
		}
		if varty == nil && validx > 0 {
			varty = vp1stty
		}
		if varty == nil && strings.HasPrefix(types.ExprString(spec.Values[0]), "C.") {
			varty = types.Typ[types.UntypedInt]
		}
		if varty == nil {
			varty = types.Typ[types.UntypedInt]
			log.Println("todo", spec.Values[idx], types.ExprString(spec.Values[idx]))
		}
		if varty == nil {
			panic("ddd")
		}
		if validx == 0 {
			vp1stty = varty
		}
		if len(spec.Values) > 0 {
			vp1stval = spec.Values[0]
			vp1stidx = validx
		}

		log.Println(varty, varname, reflect.TypeOf(varty))
		c.clinema(spec)
		vartystr := c.exprTypeNameImpl2(scope, varty, varname)
		c.out(gopp.IfElseStr(isglobvar, "static", "")).outsp()
		c.out(gopp.IfElseStr(isconst, "const", "")).outsp()
		if strings.HasPrefix(varty.String(), "untyped ") {
			c.out(sign2rety(varty.String())).outsp()
		} else {
			c.out("/*var*/").outsp()
			if isstrty2(varty) {
				c.out("cxstring*")
			} else if isarrayty2(varty) || isslicety2(varty) {
				c.out("cxarray2*")
			} else if ismapty2(varty) {
				c.out("HashTable*")
			} else if ischanty2(varty) {
				c.out("voidptr")
			} else if strings.HasPrefix(vartystr, "C_struct_") {
				c.out(vartystr[2:])
			} else if strings.Contains(vartystr, "func(") {
				tyname := tmpvarname()
				c.outf("typedef void*(*%s)()", tyname).outfh().outnl()
				c.out(tyname)
			} else {
				if isinvalidty(vartystr) {
					if len(spec.Values) == 1 {
						val0 := spec.Values[0]
						if idt, ok := val0.(*ast.Ident); ok {
							if strings.HasPrefix(idt.Name, "gxtv") {
								c.outf("__typeof__(%v)", idt.Name)
							} else {
								log.Panicln("unexpected")
							}
						} else {
							log.Panicln("unexpected")
						}
					} else {
						// log.Panicln("unexpected", len(spec.Values))
						c.out(vartystr)
					}
				} else {
					c.out(vartystr)
				}
			}
			c.outsp()
		}
		c.out(gopp.IfElseStr(isglobvar, c.pkgpfx(), ""))
		c.out(varname.Name)
		c.outsp().outeq().outsp()

		if idx < len(spec.Values) {
			c.valnames[spec.Values[idx]] = varname
			scope = putscope(scope, ast.Var, "varname", varname)
			if isglobvar && (isstrty2(varty) || isslicety2(varty) ||
				isarrayty2(varty) || isstructty2(varty) || ismapty2(varty)) {
				c.out("{0}")
			} else {
				c.genExpr(scope, spec.Values[idx])
			}
		} else {
			if isconst {
				c.out("(")
				c.genExpr(scope, vp1stval)
				c.out(")")
				c.outf("+%d", validx)
				c.outf("-%d", vp1stidx)
			} else if isglobvar {
				c.out("{0} /* 111 */") // must constant for c
			} else if isstrty2(varty) {
				c.out("cxstring_new()")
			} else if isslicety2(varty) {
				var elemsz interface{} = unsafe.Sizeof(uintptr(0))
				elemsz = gopp.IfElse(elemsz == uintptr(0), "sizeof(voidptr)", elemsz)
				c.outf("cxarray2_new(1, %v)", elemsz)
			} else if ismapty2(varty) {
				c.out("cxhashtable_new()")
			} else if isstructty2(varty) {
				tystr := sign2rety(varty.String())
				tystr = strings.Trim(tystr, "*")
				c.outf("%s_new_zero()", tystr)
			} else {
				c.out("{0} /* 222 */")
			}
		}

		if isglobvar {
			c.outfh().outnl()
		}
	}

}

func (c *g2nc) genInitGlobvars(scope *ast.Scope, pkg *ast.Package) {
	c.outf("void %sglobvars_init() {", c.pkgpfx()).outnl()
	for _, varx := range c.psctx.globvars {
		varo := varx.(*ast.ValueSpec)
		log.Println(varo.Type, reflect.TypeOf(varo.Type))
		for idx, name := range varo.Names {
			_ = idx
			keepon := false
			gotyx := c.info.TypeOf(name)
			log.Println(gotyx)
			switch goty := gotyx.(type) {
			case *types.Basic:
				if isstrty2(goty) {
					keepon = true
				}
			case *types.Array, *types.Slice, *types.Map:
				keepon = true
			default:
				gopp.G_USED(goty)
			}
			if !keepon {
				continue
			}
			c.out("//", c.exprpos(name).String()).outnl()
			assigno := &ast.AssignStmt{}
			assigno.Tok = token.ASSIGN
			assigno.Lhs = []ast.Expr{name}
			if idx < len(varo.Values) {
				assigno.Rhs = []ast.Expr{varo.Values[idx]}
			}
			c.genAssignStmt(scope, assigno)
			c.outfh().outnl()
		}
	}
	c.out("}").outnl()
}

func (this *g2nc) outsp() *g2nc   { return this.out(" ") }
func (this *g2nc) outeq() *g2nc   { return this.out("=") }
func (this *g2nc) outstar() *g2nc { return this.out("*") }
func (this *g2nc) outfh() *g2nc   { return this.out(";") }
func (this *g2nc) outnl() *g2nc   { return this.out("\n") }
func (this *g2nc) out(ss ...string) *g2nc {
	for _, s := range ss {
		// fmt.Print(s, " ")
		this.sb.WriteString(s + "")
	}
	return this
}
func (this *g2nc) outf(format string, args ...interface{}) *g2nc {
	s := fmt.Sprintf(format, args...)
	this.out(s)
	return this
}
func (this *g2nc) clinema(e ast.Node) *g2nc {
	poso := this.exprpos(e) // file:row:col
	fields := strings.Split(poso.String(), ":")
	this.outf("// #line %s %s", fields[1], fields[0]).outnl()
	return this
}

// TODO fix by typedef order
func (this *g2nc) genPrecgodefs() string {
	precgodefs := `
typedef void _Ctype_void;
typedef int32 _Ctype_int;
typedef int64 _Ctype_long;
typedef uint64 _Ctype_ulong;
typedef uint32 _Ctype_uint;
typedef int8 _Ctype_char;
typedef float32 _Ctype_float;
typedef _Ctype_long _Ctype_ptrdiff_t;
typedef float f32;
typedef double f64;
typedef uint64_t u64;
typedef int64_t i64;
typedef uintptr_t usize;
// typedef void* error;
typedef void* voidptr;
typedef char* byteptr;
`
	return precgodefs
}

func (c *g2nc) genBuiltinTypesMetatype() string {
	s := "#include <stdalign.h>\n"
	s += "#include <cxrtbase.h>\n"
	bitypes := append(types.Typ, types.TypeAlias()...)
	for idx, bityp := range bitypes {
		tyname := bityp.Name()
		if strings.Contains(tyname, " ") {
			// untyped
			continue
		}
		if strings.HasPrefix(tyname, "complex") {
			continue
		}
		if tyname == "Pointer" {
			continue
		}

		// tyname = gopp.IfElseStr(tyname == "string", "charptr", tyname)
		s += fmt.Sprintf("static const _metatype %s_metatype = {", tyname)
		s += fmt.Sprintf(".kind = %d,\n", bityp.Kind())
		if tyname == "string" {
			s += fmt.Sprintf(".size = sizeof(%s),\n", "charptr")
			s += fmt.Sprintf(".size = alignof(%s),\n", "charptr")
		} else {
			s += fmt.Sprintf(".size = sizeof(%s),\n", tyname)
			s += fmt.Sprintf(".size = alignof(%s),\n", tyname)
		}
		s += fmt.Sprintf(".tystr = \"%s\"\n", tyname)
		s += fmt.Sprintf("}; // %d\n", idx)
	}

	s += "\n"
	return s
}

func (this *g2nc) code() (string, string) {
	code := ""
	code += fmt.Sprintf("// %s of %s\n", this.psctx.bdpkgs.Dir, this.psctx.wkdir)
	code += this.psctx.ccode
	code += "/* fakec defs for " + this.psctx.bdpkgs.Dir + "\n" +
		this.psctx.fcdefscc + "\n*/\n\n"
	code += "#include <stddef.h>\n"
	code += "#include <stdalign.h>\n"
	code += "#include <cxrtbase.h>\n\n"
	code += this.genPrecgodefs() + "\n"
	code += this.sb.String()
	return code, "c"
}
