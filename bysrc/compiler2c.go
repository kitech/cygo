package main

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"gopp"
	"log"
	"os"
	"reflect"
	"runtime/debug"
	"strings"

	"github.com/thoas/go-funk"
	"golang.org/x/tools/go/ast/astutil"
	"golang.org/x/tools/go/packages"
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
	// pkgo   *ast.Package
	pkgo *packages.Package

	info  *types.Info
	scope *ast.Scope
	outfp *os.File
}

func (this *g2nc) genpkgs() {
	this.info = this.psctx.info
	this.scope = ast.NewScope(nil)
	outfp, err := os.OpenFile("/tmp/bysrc.out.c", os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0644)
	gopp.ErrPrint(err)
	this.outfp = outfp

	// pkgs order?
	for pname, pkg := range this.psctx.pkgs {
		// pkg.Scope = ast.NewScope(nil)
		this.curpkg = pkg.Name
		this.pkgo = pkg

		gopp.G_USED(pname)

		// this.genpkg(pname, pkg)
		// this.calcClosureInfo(pkg.Scope, pkg)
		// this.calcDeferInfo(pkg.Scope, pkg)
		// this.genGostmtTypes(pkg.Scope, pkg)
		// this.genChanTypes(pkg.Scope, pkg)
		// this.genMultiretTypes(pkg.Scope, pkg)
		// this.genFuncs(pkg)
	}

	scope := this.scope
	for pname, pkg := range this.psctx.pkgs {
		log.Println(pname, pkg)
		this.genpkg2(pname, pkg)
		this.calcClosureInfo(scope, pkg)
		this.calcDeferInfo(scope, pkg)
		this.genGostmtTypes(scope, pkg)
		this.genChanTypes(scope, pkg)
		this.genMultiretTypes(scope, pkg)
		this.genFuncs(pkg)
	}

}
func (c *g2nc) pkgpfx() string {
	pfx := ""
	if c.curpkg == "main" {
		pfx = c.curpkg + "_"
	} else {
		if c.psctx.pkgrename != "" {
			pfx = c.psctx.pkgrename
		} else {
			pfx = c.curpkg
		}
		pfx += "_"
	}
	return pfx
}

func (this *g2nc) genpkg2(name1 string, pkg *packages.Package) {
	log.Println("processing package", name1, pkg.GoFiles, len(pkg.Syntax))
	log.Println("pkg other files", name1, pkg.ExportFile, pkg.OtherFiles, pkg.CompiledGoFiles)
	for _, f := range pkg.Syntax {
		astutil.Apply(f, func(c *astutil.Cursor) bool {
			// log.Println(name1, c.Node())
			return true
		}, func(c *astutil.Cursor) bool {
			return true
		})
		this.genfile(this.scope, f.Name.Name, f)
	}
}

func (this *g2nc) genfile(scope *ast.Scope, name string, f *ast.File) {
	log.Println("processing", name, f.Name, exprpos(this.psctx, f))
	this.outf("// mod %s %s ", name, exprpos(this.psctx, f)).outnl()
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
func (c *g2nc) calcClosureInfo(scope *ast.Scope, pkg *packages.Package) {
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
func (c *g2nc) calcDeferInfo(scope *ast.Scope, pkg *packages.Package) {
	c.out("// defer types ", fmt.Sprintf("%d", len(c.psctx.defers))).outnl()
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
func (c *g2nc) genGostmtTypes(scope *ast.Scope, pkg *packages.Package) {
	c.out("// gostmt types ", fmt.Sprintf("%d", len(c.psctx.gostmts))).outnl()
	for idx, gostmt := range c.psctx.gostmts {
		c.outf("// %d %v %v", idx, gostmt.Call.Fun, gostmt.Call.Args).outnl()
		c.genFiberStargs(scope, gostmt.Call)
		c.outnl()
	}
}
func (c *g2nc) genChanTypes(scope *ast.Scope, pkg *packages.Package) {
	c.out("// chan types ", fmt.Sprintf("%d", len(c.psctx.chanops))).outnl()
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
func (c *g2nc) genMultiretTypes(scope *ast.Scope, pkg *packages.Package) {
	c.out("// multirets types ", fmt.Sprintf("%d", len(c.psctx.gostmts))).outnl()
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

func (this *g2nc) genFuncs(pkg *packages.Package) {
	this.out("// funcs decl ", fmt.Sprintf("%d", len(this.psctx.funcDeclsv))).outnl()
	scope := this.scope
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

	this.genInitGlobvars(this.scope, pkg)

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
func (c *g2nc) genPostFuncDecl(scope *ast.Scope, d *ast.FuncDecl) {
	// gen fiber wrapper funcs
	for _, gostmt := range c.psctx.gostmts {
		// how compare called func is current func
		fe := gostmt.Call.Fun
		mat := false
		switch te := fe.(type) {
		case *ast.Ident:
			mat = te.Name == d.Name.Name
		default:
			log.Println("todo", fe, reflect.TypeOf(fe))
		}
		if mat {
			c.genFiberStwrap(scope, gostmt.Call)
		}
	}
}
func (this *g2nc) genFuncDecl(scope *ast.Scope, fd *ast.FuncDecl) {
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
		this.out(recvtystr + "_" + fd.Name.String())
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
			for idx2, name := range arge.Names {
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
			this.out("Array* deferarr = cxarray_new()").outfh().outnl()
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
	c.out("cxrt_init_env()").outfh().outnl()
	c.out("// TODO arguments populate").outnl()
	c.out("// globvars populate").outnl()
	c.outf("%sglobvars_init()", c.pkgpfx()).outfh().outnl()
	c.out("// all func init()").outnl()
	c.outf("%sinit()", c.pkgpfx()).outfh().outnl()
	c.out("main_main()").outfh().outnl()
	c.out("return 0").outfh().outnl()
	c.out("}").outnl()
}
func (c *g2nc) genInitFuncs(scope *ast.Scope, pkg *packages.Package) {
	for idx, fd := range c.psctx.initFuncs {
		c.outf("// %s", c.exprpos(fd).String()).outnl()
		c.out("static").outsp()
		c.outf("void %sinit_%d()", c.pkgpfx(), idx)
		c.genBlockStmt(scope, fd.Body)
	}
	c.outf("void %sinit(){", c.pkgpfx()).outnl()
	for idx, _ := range c.psctx.initFuncs {
		c.outf("%sinit_%d()", c.pkgpfx(), idx).outfh().outnl()
	}
	c.out("}").outnl().outnl()
}

func (this *g2nc) genBlockStmt(scope *ast.Scope, stmt *ast.BlockStmt) {
	this.out("{").outnl()
	scope = ast.NewScope(scope)
	for idx, s := range stmt.List {
		this.genStmt(scope, s, idx)
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
			this.outnl()
			this.outf("#line %s \"%s\"", fields[1], fields[0]).outnl()
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
				c.out(te.(*ast.Ident).Name).outeq()
				c.out(tvname).out("->").out(tmpvarname2(idx)).outfh().outnl()
			}
			c.outf("cxfree(%s)", tvname).outfh().outnl()
		} else if iserrorty2(c.info.TypeOf(s.Lhs[i])) {
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
	c.out(funame+"_fiber", "(void* vpargs)").outnl()
	c.out("{").outnl()
	c.out(stname, "*args = (", stname, "*)vpargs").outfh().outnl()
	c.out(gopp.IfElseStr(pkgo == nil, "", pkgo.Name()+"_"))
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
	c.out("for (")

	c.genStmt(scope, s.Init, 0)
	// c.out(";") // TODO ast.AssignStmt has put ;
	if isefor {
		// c.out(";")
	}
	c.genExpr(scope, s.Cond)
	c.out(";")

	c.out(")")
	c.out("{")
	c.genBlockStmt(scope, s.Body)
	c.genStmt(scope, s.Post, 2) // Post move to real post, to resolve ';' problem
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
		c.outf("  for (int %s = 0; %s < array_size(%v); %s++) {",
			keyidstr, keyidstr, s.X, keyidstr).outnl()
		if s.Value != nil {
			valtystr := c.exprTypeName(scope, s.Value)
			c.outf("     %s %v = {0}", valtystr, s.Value).outfh().outnl()
			var tmpvar = tmpvarname()
			c.outf("    void* %s = {0}", tmpvar).outfh().outnl()
			c.outf("    int rv = array_get_at(%v, %s, (void**)&%s)", s.X, keyidstr, tmpvar).outfh().outnl()
			c.outf("    %v = (%v)(uintptr_t)%s", s.Value, valtystr, tmpvar).outfh().outnl()
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
	c.out(s.Tok.String())
	if s.Label != nil {
		c.out(s.Label.Name)
	}
	c.outfh().outnl()
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
			c.genSwitchStmtNum(scope, s)
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

func (c *g2nc) genCallExpr(scope *ast.Scope, te *ast.CallExpr) {
	// log.Println(te, te.Fun, reflect.TypeOf(te.Fun))
	scope = putscope(scope, ast.Fun, "infncall", te.Fun)
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
			} else {
				c.genCallExprNorm(scope, te)
			}
		}
	case *ast.SelectorExpr:
		if c.funcistype(te.Fun) {
			c.genTypeCtor(scope, te)
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
			c.outf("array_size(")
			c.genExpr(scope, arg0)
			c.out(")")
		} else if funame == "cap" {
			c.out("array_capacity(")
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
			c.outf("cxarray_append(")
			c.genExpr(scope, arg0)
			c.out(",")
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
		c.outf(", (void*)(uintptr_t)%s, 0)", keystr).outfh().outnl()
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
	c.out("(")
	c.out("__FILE__, __LINE__, __func__",
		gopp.IfElseStr(len(te.Args) > 0, ",", "")).outnl()
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
func (c *g2nc) genCallExprNorm(scope *ast.Scope, te *ast.CallExpr) {
	// funame := te.Fun.(*ast.Ident).Name

	selfn, isselfn := te.Fun.(*ast.SelectorExpr)
	isidt := false
	iscfn := false
	ispkgsel := false
	isifacesel := false
	if isselfn {
		var selidt *ast.Ident
		selidt, isidt = selfn.X.(*ast.Ident)
		iscfn = isidt && selidt.Name == "C"
		selty := c.info.TypeOf(selfn.X)
		ispkgsel = isinvalidty2(selty)

		selxty := c.info.TypeOf(selfn.X)
		switch ne := selxty.(type) {
		case *types.Named:
			isifacesel = isiface2(ne.Underlying())
		}
	}
	gotyx := c.info.TypeOf(te.Fun)
	isvardic := false
	var goty *types.Signature
	if gotyx != nil {
		goty = gotyx.(*types.Signature)
		isvardic = goty.Variadic()
	}

	// log.Println(te.Args, te.Fun, gotyx, reflect.TypeOf(gotyx), goty.Variadic())
	haslv := c.psctx.kvpairs[te] != nil

	idt := newIdent(tmpvarname())
	if isvardic && haslv {
		c.out("{0}").outfh().outnl()
		c.outf("Array* %s = cxarray_new()", idt.Name).outfh().outnl()
		for idx, e1 := range te.Args {
			if idx < goty.Params().Len()-1 {
				continue
			}
			c.outf("array_add(%s, (void*)", idt.Name)
			c.genExpr(scope, e1)
			c.out(")")
			c.outfh().outnl()
		}
		c.genExpr(scope, c.psctx.kvpairs[te].(ast.Expr))
		c.outeq()
	}

	if isselfn {
		if iscfn {
			c.genExpr(scope, selfn.Sel)
		} else if isifacesel {
			c.genExpr(scope, selfn.X)
			c.out("->")
			c.genExpr(scope, selfn.Sel)
		} else {
			vartystr := c.exprTypeName(scope, selfn.X)
			vartystr = strings.TrimRight(vartystr, "*")
			c.out(vartystr + "_" + selfn.Sel.Name)
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
	if isselfn && !iscfn && !ispkgsel {
		c.genExpr(scope, selfn.X)
		c.out(gopp.IfElseStr(isifacesel, "->data", ""))
		c.out(gopp.IfElseStr(len(te.Args) > 0, ",", ""))
	}
	for idx, e1 := range te.Args {
		if isvardic && idx == goty.Params().Len()-1 {
			c.out(idt.Name)
			break
		}
		c.genExpr(scope, e1)
		c.out(gopp.IfElseStr(idx == len(te.Args)-1, "", ", "))
	}
	c.out(")")

	// check if real need, ;\n
	cs := c.psctx.cursors[te]
	if cs.Name() != "Args" {
		// c.outfh().outnl()
	}
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
				} else {
					c.outf("cxstring_new_char(%v)", ce.Name)
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
	c.out("void* rvx = cxrt_chan_recv(")
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
func (c *g2nc) genDeferStmtSet(scope *ast.Scope, e *ast.DeferStmt) {
	deferi := c.getdeferinfo(e)
	c.outf("cxarray_append(deferarr, (void*)%d)", deferi.idx)
}
func (c *g2nc) genDeferStmt(scope *ast.Scope, e *ast.ReturnStmt) {
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
	if len(defers) == 0 {
		return
	}
	c.out("// defer section").outnl()
	c.out("{").outnl()
	c.out("int deferarrsz = array_size(deferarr)").outfh().outnl()
	c.out("for (int deferarri = deferarrsz-1; deferarri>=0; deferarri--)")
	c.out("{").outnl()
	c.out("uintptr_t deferarrn = 0").outfh().outnl()
	c.out("array_get_at(deferarr, deferarri, (void**)&deferarrn)").outfh().outnl()
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

func (this *g2nc) genExpr(scope *ast.Scope, e ast.Expr) {
	varname := scope.Lookup("varname")
	if varname != nil {
		vartyp := this.info.TypeOf(varname.Data.(ast.Expr))
		log.Println(vartyp, varname)
		if iseface2(vartyp) {
			_, iscallexpr := e.(*ast.CallExpr)
			_, isidt := e.(*ast.Ident)
			// _, lisidt := varname.Data.(ast.Expr).(*ast.Ident)
			if !iscallexpr && !isidt {
				// vartyp2 := reflect.TypeOf(varname.Data.(ast.Expr))
				this.out("(cxeface){0}").outfh().outnl()

				tmpvar := tmpvarname()
				this.out(this.exprTypeName(scope, e), tmpvar, "=")
				ns := putscope(scope, ast.Var, "varname", newIdent(tmpvar))
				this.genExpr2(ns, e)
				this.outfh().outnl()
				this.genExpr2(scope, varname.Data.(ast.Expr))
				this.outeq()
				this.outf("cxeface_new_of2((void*)&%s, sizeof(%s))", tmpvar, tmpvar)
				return
			}
		}
	}
	this.genExpr2(scope, e)
}
func (this *g2nc) genExpr2(scope *ast.Scope, e ast.Expr) {
	// log.Println(reflect.TypeOf(e))
	switch te := e.(type) {
	case *ast.Ident:
		idname := te.Name
		idname = gopp.IfElseStr(idname == "nil", "nilptr", idname)
		idname = gopp.IfElseStr(idname == "string", "cxstring*", idname)
		eobj := this.info.ObjectOf(te)
		log.Println(e, eobj, isglobalid(this.psctx, te))
		if eobj != nil {
			pkgo := eobj.Pkg()
			if pkgo != nil {
				// this.out(pkgo.Name())
			}
		}
		if strings.HasPrefix(idname, "_Cfunc_") || isglobalid(this.psctx, te) {
			this.out(this.pkgpfx())
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
			if vo == nil {
				// vo = lookscope(scope, "varname")
			}
			if vo == nil {
				gotyval := this.info.Types[te]
				log.Println("temp var?", vo, this.exprpos(te), gotyval)
			}
			for idx, ex := range te.Elts {
				switch be := ex.(type) {
				case *ast.KeyValueExpr:
					// log.Println(vo == nil, ex, idx, this.exprpos(ex))
					this.genCxmapAddkv(scope, vo.Data, be.Key, be.Value)
					this.outfh().outnl()
				default:
					log.Println("unknown", idx, reflect.TypeOf(ex))
				}
			}
		case *ast.ArrayType:
			var vo = scope.Lookup("varname")
			if vo == nil {
				// vo = lookscope(scope, "varname")
			}
			if vo == nil {
				gotyval := this.info.Types[te]
				log.Println("temp var?", vo, this.exprpos(te), gotyval)
			}
			this.outf("cxarray_new()").outfh().outnl()
			for idx, ex := range te.Elts {
				// log.Println(vo == nil, ex, idx, this.exprpos(ex))
				this.genCxarrAdd(scope, vo.Data, ex, idx)
				this.outfh().outnl()
			}
			if be == nil {
			}
		case *ast.Ident: // TODO
			var vo = scope.Lookup("varname")
			this.outf("%v_new_zero()", this.exprTypeName(scope, be)).outfh().outnl()
			for idx, ex := range te.Elts {
				this.genExpr(scope, vo.Data.(ast.Expr))
				this.out("->")
				switch ef := ex.(type) {
				case *ast.KeyValueExpr:
					this.genExpr(scope, ef.Key)
					this.outeq()
					this.genExpr(scope, ef.Value)
				default:
					tetyx := this.info.TypeOf(te)
					switch tetyx.(type) {
					case *types.Named:
						tetyx = tetyx.Underlying()
					default:
						log.Panicln("need more", tetyx, reflect.TypeOf(tetyx))
					}
					tety := tetyx.(*types.Struct)
					this.out(tety.Field(idx).Name())
					this.outeq()
					this.genExpr(scope, ex)
				}
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
				log.Println("unknown", t.String())
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
		this.out("void*")
	case *ast.IndexExpr:
		varty := this.info.TypeOf(te.X)
		vo := scope.Lookup("varval")
		if ismapty(varty.String()) {
			if vo == nil {
				this.genCxmapAddkv(scope, te.X, te.Index, nil)
			} else {
				this.genCxmapAddkv(scope, te.X, te.Index, vo.Data)
			}
		} else if isslicety(varty.String()) {
			// get or set?
			if vo == nil { // right value
				this.genCxarrGet(scope, te.X, te.Index)
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
		} else {
			log.Println("todo", te.X, te.Index)
			log.Println("todo", reflect.TypeOf(te.X), varty.String(), this.exprpos(te.X))
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
			this.outf("cxarray_slice(%v, ", te.X)
			this.genExpr(scope, lowe)
			this.out(",")

			if highe == nil {
				this.outf("array_size(%v)", te.X)
			} else {
				this.genExpr(scope, te.High)
			}
			this.out(")")
		} else {
			log.Println("todo", varty, te)
		}
	case *ast.SelectorExpr:
		if iscsel(te.X) {
		} else {
			this.genExpr(scope, te.X)
			selxty := this.info.TypeOf(te.X)
			log.Println(selxty, reflect.TypeOf(selxty))
			if isinvalidty2(selxty) { // package
				this.out("_")
			} else {
				this.out("->")
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

	c.outf("hashtable_add(%v, (void*)(uintptr_t)%v, (void*)(uintptr_t)%s)",
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
	log.Println(varobj == nil, vnamex, ve, idx)
	pkgpfx := ""
	if isglobalid(c.psctx, vnamex.(*ast.Ident)) {
		pkgpfx = varobj.Pkg().Name()
	}
	pkgpfx = gopp.IfElseStr(pkgpfx == "", "", pkgpfx+"_")
	c.outf("array_add(%s%v, (void*)(uintptr_t)%v)", pkgpfx,
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

	c.outf("array_replace_at(%v, (void*)(uintptr_t)%v, %v, nilptr)",
		vname, valstr, idxstr).outfh().outnl()
}
func (c *g2nc) genCxarrGet(scope *ast.Scope, vname ast.Expr, vidx ast.Expr) {
	idxstr := ""

	switch te := vidx.(type) {
	case *ast.BasicLit:
		idxstr = te.Value
	case *ast.Ident:
		idxstr = te.Name
	default:
		log.Println("todo", vidx, reflect.TypeOf(vidx))
	}

	c.outf("cxarray_get_at(%v, %v)", vname, idxstr)
}
func (this *g2nc) exprTypeName(scope *ast.Scope, e ast.Expr) string {
	// log.Println(e, reflect.TypeOf(e))
	tyname := this.exprTypeNameImpl(scope, e)
	if tyname == "unknownty" {
		// log.Panicln(tyname, e, reflect.TypeOf(e), this.exprpos(e))
	}
	return tyname
}
func (this *g2nc) exprTypeNameImpl(scope *ast.Scope, e ast.Expr) string {

	{
		// return "unknownty"
	}

	goty := this.info.TypeOf(e)
	if goty == nil {
		log.Println(this.exprstr(e))
		log.Panicln(e, this.exprpos(e), reflect.TypeOf(e))
	}
	val := this.exprTypeNameImpl2(scope, goty, e)
	if isinvalidty(val) {
		// log.Panicln("unreachable")
		val = this.exprstr(e)
		val = strings.Replace(val, "C.", "", 1)
		log.Println(val)
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
	// log.Println(goty, reflect.TypeOf(goty), e, reflect.TypeOf(e))

	switch te := goty.(type) {
	case *types.Basic:
		if isstrty(te.Name()) {
			return "cxstring*"
		} else {
			if strings.Contains(te.Name(), "string") {
				log.Println(te.Name())
			}
			// log.Println(te, reflect.TypeOf(e), te.Info(), te.Name(), te.Underlying(), reflect.TypeOf(te.Underlying()))
			return strings.Replace(te.String(), ".", "_", 1)
			return te.Name()
		}
	case *types.Named:
		teobj := te.Obj()
		pkgo := teobj.Pkg()
		undty := te.Underlying()
		// log.Println(teobj, pkgo, undty)
		switch ne := undty.(type) {
		case *types.Interface:
			if pkgo == nil { // builtin???
				return teobj.Name() + "*"
			}
			return fmt.Sprintf("%s_%s", pkgo.Name(), teobj.Name())
		case *types.Struct:
			tyname := teobj.Name()
			if strings.HasPrefix(tyname, "_Ctype_") {
				return fmt.Sprintf("%s_%s", pkgo.Name(), tyname[7:])
			}
			return fmt.Sprintf("%s_%s", pkgo.Name(), teobj.Name())
		case *types.Basic:
			tyname := teobj.Name()
			if strings.HasPrefix(tyname, "_Ctype_") {
				return tyname[7:]
			}
			return fmt.Sprintf("%s_%s", pkgo.Name(), teobj.Name())
		case *types.Array:
			tyname := teobj.Name()
			if strings.HasPrefix(tyname, "_Ctype_") {
				return tyname[7:]
			}
		default:
			gopp.G_USED(ne)
		}
		log.Println("todo", teobj.Name(), reflect.TypeOf(undty), goty)
		return "todo" + teobj.Name()
		// return sign2rety(te.String())
	case *types.Pointer:
		tystr := this.exprTypeNameImpl2(scope, te.Elem(), e)
		tystr += "*"
		// log.Println(tystr)
		return tystr
	case *types.Slice, *types.Array:
		tystr := te.String()
		if tystr == "[0]byte" {
			return "void"
		}
		return "Array*"
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
		log.Println("todo", goty, reflect.TypeOf(goty), isudty, tyval, te, this.exprpos(e))
		return te.String()
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
			default:
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
			log.Println("todo", reflect.TypeOf(d), reflect.TypeOf(spec), tspec.Path, tspec.Name)
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
					this.outf("obj->%s = cxarray_new()", fldname.Name).outfh().outnl()
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

// orignal Lookup igore Outer scope, but this not
func lookscope(scope *ast.Scope, name string) *ast.Object {
	if scope == nil {
		return nil
	}
	obj := scope.Lookup(name)
	if obj != nil {
		return obj
	}
	return lookscope(scope.Outer, name)
}

var vp1stval ast.Expr
var vp1stty types.Type
var vp1stidx int

func (c *g2nc) genValueSpec(scope *ast.Scope, spec *ast.ValueSpec, validx int) {
	for idx, varname := range spec.Names {
		isglobvar := c.psctx.isglobal(varname)
		varty := c.info.TypeOf(spec.Type)
		if varty == nil && idx < len(spec.Values) {
			varty = c.info.TypeOf(spec.Values[idx])
		}
		if varty == nil && validx > 0 {
			varty = vp1stty
		}
		// log.Println(validx, idx, varname, varty, spec.Values, len(c.info.Types))
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
		vartystr := c.exprTypeNameImpl2(scope, varty, varname)
		isconst := false
		c.out(gopp.IfElseStr(isglobvar, "static", "")).outsp()
		if strings.HasPrefix(varty.String(), "untyped ") {
			isconst = true
			c.out("/*const*/").outsp()
			c.out(sign2rety(varty.String())).outsp()
		} else {
			c.out("/*var*/").outsp()
			if isstrty2(varty) {
				c.out("cxstring*")
			} else if isarrayty2(varty) || isslicety2(varty) {
				c.out("Array*")
			} else if ismapty2(varty) {
				c.out("HashTable*")
			} else if ischanty2(varty) {
				c.out("voidptr")
			} else {
				c.out(vartystr)
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
				c.out("cxarray_new()")
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

func (c *g2nc) genInitGlobvars(scope *ast.Scope, pkg *packages.Package) {
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
			assigno.Tok = token.EQL
			assigno.Lhs = []ast.Expr{name}
			assigno.Rhs = []ast.Expr{varo.Values[idx]}
			c.genAssignStmt(scope, assigno)
			c.outfh().outnl()
		}
	}
	c.out("}")
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
		this.outfp.WriteString(s + "")
	}
	return this
}
func (this *g2nc) outf(format string, args ...interface{}) *g2nc {
	s := fmt.Sprintf(format, args...)
	this.out(s)
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
`
	return precgodefs
}

func (this *g2nc) code() (string, string) {
	code := ""
	code += fmt.Sprintf("// %s of %s\n", this.psctx.path, this.psctx.wkdir)
	code += this.psctx.ccode
	code += "#include <cxrtbase.h>\n\n"
	code += this.genPrecgodefs() + "\n"
	code += this.sb.String()

	log.Println(code)
	return code, "c"
}
