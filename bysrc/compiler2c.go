package main

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"gopp"
	"log"
	"reflect"
	"runtime/debug"
	"strings"
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
		pkg.Scope = ast.NewScope(nil)
		this.curpkg = pkg.Name
		this.pkgo = pkg

		this.genpkg(pname, pkg)
		this.calcClosureInfo(pkg.Scope, pkg)
		this.genGostmtTypes(pkg.Scope, pkg)
		this.genChanTypes(pkg.Scope, pkg)
		this.genFuncs(pkg)
	}

}
func (c *g2nc) pkgpfx() string {
	pfx := ""
	if c.curpkg == "main" {
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
func (c *g2nc) genGostmtTypes(scope *ast.Scope, pkg *ast.Package) {
	c.out("// gostmt types", fmt.Sprintf("%d", len(c.psctx.gostmts))).outnl()
	for idx, gostmt := range c.psctx.gostmts {
		c.outf("// %d %v %v", idx, gostmt.Call.Fun, gostmt.Call.Args).outnl()
		c.genFiberStargs(scope, gostmt.Call)
		c.outnl()
	}
}
func (c *g2nc) genChanTypes(scope *ast.Scope, pkg *ast.Package) {
	c.out("// chan types", fmt.Sprintf("%d", len(c.psctx.chanops))).outnl()
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
func (this *g2nc) genFuncs(pkg *ast.Package) {
	scope := pkg.Scope
	// ordered funcDeclsv
	for _, d := range this.psctx.funcDeclsv {
		if d == nil {
			log.Println("wtf", d)
			continue
		}
		this.genDecl(scope, d)
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

	pkgpfx := this.pkgpfx()
	this.genFieldList(scope, fd.Type.Results, true, false, "", false)
	this.outsp()
	if fd.Recv != nil {
		recvtystr := this.exprTypeName(scope, fd.Recv.List[0].Type)
		recvtystr = strings.TrimRight(recvtystr, "*")
		this.out(pkgpfx + recvtystr + "_" + fd.Name.String())
	} else {
		this.out(pkgpfx + fd.Name.String())
	}
	this.out("(")
	if fd.Recv != nil {
		this.genFieldList(scope, fd.Recv, false, true, ",", true)
		if fd.Type.Params != nil && fd.Type.Params.NumFields() > 0 {
			this.out(",")
		}
	}
	if fd.Name.Name == "main" {
		this.out("int argc, char**argv")
	}
	this.genFieldList(scope, fd.Type.Params, false, true, ",", true)
	this.out(")").outnl()
	if fd.Body != nil {
		scope = ast.NewScope(scope)
		scope.Insert(ast.NewObj(ast.Fun, fd.Name.Name))
		this.genBlockStmt(scope, fd.Body)
	} else {
		this.outfh()
	}
	this.outnl()
}

func (this *g2nc) genBlockStmt(scope *ast.Scope, stmt *ast.BlockStmt) {
	this.out("{").outnl()
	if scope.Lookup("main") != nil {
		this.out("{ cxrt_init_env(); }").outnl()
		this.out("{ \n // func init() \n }").outnl()
	}
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
		this.out("// ", posinfo).outnl()
		stmtstr := this.prtnode(stmt)
		if !strings.ContainsAny(strings.TrimSpace(stmtstr), "\n") {
			this.out("// ").outnl()
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
		} else {
			if s.Tok.String() == ":=" {
				c.out(c.exprTypeName(scope, s.Rhs[i])).outsp()
			}
			c.genExpr(scope, s.Lhs[i])

			var ns = putscope(scope, ast.Var, "varname", s.Lhs[i])
			c.out(" = ")
			c.genExpr(ns, s.Rhs[i])
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

	stname := funame + "_fiber_args"
	c.out("static").outsp()
	c.out("void").outsp()
	c.out(funame+"_fiber", "(void* vpargs)").outnl()
	c.out("{").outnl()
	c.out(stname, "*args = (", stname, "*)vpargs").outfh().outnl()
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

	c.out("// gogorun", funame).outnl()
	c.out("{")
	c.outf("%s* args = (%s*)cxmalloc(sizeof(%s))", stname, stname, stname).outfh().outnl()
	for idx, arg := range e.Args {
		c.outf("args->a%d", idx).outeq()
		c.genExpr(scope, arg)
		c.outfh().outnl()
	}
	c.outf("cxrt_fiber_post(%s, args)", wfname).outfh().outnl()
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
	log.Println(te, te.Fun, reflect.TypeOf(te.Fun))
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
				log.Println("gen clos call")
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
		c.genExpr(scope, be.X)
		c.out(")")
		c.out("(")
		log.Println(c.exprstr(te))
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
	c.out("__FILE__, __LINE__, __func__", gopp.IfElseStr(len(te.Args) > 0, ",", "")).outnl()
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
			c.out("(")
			c.out(tmpnames[idx])
			// c.genExpr(scope, e1)
			c.out(")->len,")

			c.out("(")
			c.out(tmpnames[idx])
			// c.genExpr(scope, e1)
			c.out(")->ptr")
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
	if isselfn {
		var selidt *ast.Ident
		selidt, isidt = selfn.X.(*ast.Ident)
		iscfn = isidt && selidt.Name == "C"
		selty := c.info.TypeOf(selfn.X)
		ispkgsel = isinvalidty2(selty)
	}

	if isselfn {
		if iscfn {
			c.genExpr(scope, selfn.Sel)
		} else {
			vartystr := c.exprTypeName(scope, selfn.X)
			vartystr = strings.TrimRight(vartystr, "*")
			c.out(vartystr + "_" + selfn.Sel.Name)
		}
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

	if lefte != nil {
		c.out("{0}").outfh().outnl()
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
	c.out("return").outsp()
	for idx, ae := range e.Results {
		c.genExpr(scope, ae)
		if idx < len(e.Results)-1 {
			c.out(",") // TODO
		}
	}
	// c.outfh().outnl().outnl()
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
		// log.Println(this.exprTypeName(scope, fld.Type))
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
		this.out(idname, "")
	case *ast.ArrayType:
		log.Println("unimplemented", te, reflect.TypeOf(e), e.Pos())
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
			// this.out(fmt.Sprintf("(%v*)cxmalloc(sizeof(%v))", t2.Type, t2.Type)) //.outnl()
			this.outf("%v_new_zero()", t2.Type) //.outnl()
			keepop = false
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
			this.outf("cxarray_new()").outfh().outnl()
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
		this.outf("(%s)(*(void**)(", this.exprTypeName(scope, te.Type))
		this.genExpr(scope, te.X)
		this.out(".data))")
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

	c.outf("array_add(%v, (void*)(uintptr_t)%v)",
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

	c.outf("array_replace_at(%v, (void*)(uintptr_t)%v, %v, 0)",
		vname, valstr, idxstr).outfh().outnl()
}
func (c *g2nc) genCxarrGet(scope *ast.Scope, vname ast.Expr, vidx ast.Expr) {
	idxstr := ""

	switch te := vidx.(type) {
	case *ast.BasicLit:
		idxstr = te.Value
	default:
		log.Println("todo", vidx, reflect.TypeOf(vidx))
	}

	c.outf("cxarray_get_at(%v, %v)", vname, idxstr)
}
func (this *g2nc) exprTypeName(scope *ast.Scope, e ast.Expr) string {
	log.Println(e, reflect.TypeOf(e))
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
	// log.Println(goty, reflect.TypeOf(goty))

	switch te := goty.(type) {
	case *types.Basic:
		if isstrty(te.Name()) {
			return "cxstring*"
		} else {
			if strings.Contains(te.Name(), "string") {
				log.Println(te.Name())
			}
			log.Println(te, reflect.TypeOf(e))
			return te.Name()
		}
	case *types.Named:
		return te.Obj().Name()
		// return sign2rety(te.String())
	case *types.Pointer:
		tystr := this.exprTypeNameImpl2(scope, te.Elem(), e)
		tystr += "*"
		// log.Println(tystr)
		return tystr
	case *types.Slice, *types.Array:
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
	default:
		log.Println("todo", goty, reflect.TypeOf(goty), isudty, tyval, te)
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

	switch te := goty.(type) {
	case *types.Basic:
		if isstrty(te.Name()) {
			return ".*s"
		} else {
			switch te.Kind() {
			case types.Float32, types.Float64:
				return "f"
			default:
				return "d"
			}
		}
	case *types.Named:
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
	switch te := spec.Type.(type) {
	case *ast.StructType:
		this.outf("typedef struct %s %s", spec.Name.Name, spec.Name.Name).outfh().outnl()
		this.outf("struct %s {", spec.Name.Name)
		this.outnl()
		this.genExpr(scope, spec.Type)
		this.out("}").outfh().outnl()
		this.outnl()
		this.out("static").outsp()
		this.outf("%s* %s_new_zero() {", spec.Name.Name, spec.Name.Name).outnl()
		this.outf("  %s* obj = (%s*)cxmalloc(sizeof(%s))",
			spec.Name.Name, spec.Name.Name, spec.Name.Name).outfh().outnl()
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
		this.outf("typedef %v %v", spec.Type, spec.Name.Name).outfh().outnl()
		if this.pkgpfx() != "" {
			this.outf("typedef %v %s%v", spec.Type, this.pkgpfx(), spec.Name.Name).outfh().outnl()
		}
	case *ast.StarExpr:
		this.out("typedef").outsp()
		this.genExpr(scope, te.X)
		this.out("*").outsp()
		this.out(spec.Name.Name)
		this.outfh().outnl()

		if this.pkgpfx() != "" {
			this.out("typedef").outsp()
			this.genExpr(scope, te.X)
			this.out("*").outsp()
			this.out(this.pkgpfx() + spec.Name.Name)
			this.outfh().outnl()
		}
	case *ast.SelectorExpr:
		this.out("typedef").outsp()
		this.genExpr(scope, te.X)
		this.out("_")
		this.genExpr(scope, te.Sel)
		this.outsp()
		this.out(spec.Name.Name)
		this.outfh().outnl()

		if this.pkgpfx() != "" {
			this.out("typedef").outsp()
			this.genExpr(scope, te.X)
			this.out("_")
			this.genExpr(scope, te.Sel)
			this.outsp()
			this.out(this.pkgpfx() + spec.Name.Name)
			this.outfh().outnl()
		}
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
				c.out(sign2rety(varty.String()))
			}
			c.outsp()
		}
		if isglobvar && isstrty2(varty) {
			log.Println("not supported global string/struct value", varname)
		}
		c.out(varname.Name)
		c.outsp().outeq().outsp()

		if idx < len(spec.Values) {
			var ns = putscope(scope, ast.Var, "varname", varname)
			c.genExpr(ns, spec.Values[idx])
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

func (this *g2nc) code() (string, string) {
	code := ""
	code += fmt.Sprintf("// %s of %s\n", this.psctx.bdpkgs.Dir, this.psctx.wkdir)
	code += this.psctx.ccode
	code += "#include <cxrtbase.h>\n\n"
	code += this.sb.String()
	return code, "c"
}
