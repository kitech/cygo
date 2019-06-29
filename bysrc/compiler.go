package main

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"gopp"
	"log"
	"reflect"
	"strings"

	"golang.org/x/tools/go/ast/astutil"
)

type g2nc struct {
	psctx *ParserContext
	sb    strings.Builder

	info *types.Info
}

func (c *g2nc) exprpos(e ast.Expr) token.Position {
	return c.psctx.fset.Position(e.Pos())
}

func (this *g2nc) genpkgs() {
	this.info = &this.psctx.info

	// pkgs order?
	for pname, pkg := range this.psctx.pkgs {
		pkg.Scope = ast.NewScope(nil)
		this.genpkg(pname, pkg)
	}
}

func (this *g2nc) genpkg(name string, pkg *ast.Package) {
	log.Println("processing package", name)
	for name, f := range pkg.Files {
		this.genfile(pkg.Scope, name, f)
	}
}
func (this *g2nc) genfile(scope *ast.Scope, name string, f *ast.File) {
	log.Println("processing file", name)

	// decls order?
	for _, d := range f.Decls {
		this.genDecl(scope, d)
	}
}

func (this *g2nc) genDecl(scope *ast.Scope, d ast.Decl) {
	switch td := d.(type) {
	case *ast.FuncDecl:
		this.genPreFuncDecl(scope, td)
		this.genFuncDecl(scope, td)
	case *ast.GenDecl:
		this.genGenDecl(scope, td)
	default:
		log.Println("unimplemented", reflect.TypeOf(d))
	}
}
func (this *g2nc) genPreFuncDecl(scope *ast.Scope, d *ast.FuncDecl) {
	// goroutines struct and functions wrapper
	var gostmts []*ast.GoStmt
	var sdstmts []*ast.SendStmt
	// var rvstmts []* UnaryExpr
	astutil.Apply(d, func(c *astutil.Cursor) bool {
		switch t := c.Node().(type) {
		case *ast.GoStmt:
			gostmts = append(gostmts, t)
		case *ast.SendStmt:
			sdstmts = append(sdstmts, t)
		}
		return true
	}, nil)
	if len(gostmts) > 0 {
		log.Println("found gostmts", d.Name.Name, len(gostmts))
	}
	for _, stmt := range gostmts {
		this.genRoutineStargs(scope, stmt.Call)
		this.genRoutineStwrap(scope, stmt.Call)
	}
	for _, stmt := range sdstmts {
		this.genChanStargs(scope, stmt.Chan) // chan structure args
	}
}
func (this *g2nc) genFuncDecl(scope *ast.Scope, d *ast.FuncDecl) {
	this.genFieldList(scope, d.Type.Results, true, false, "", false)
	this.out(d.Name.String())
	this.out("(")
	this.genFieldList(scope, d.Type.Params, false, true, ",", true)
	this.out(")").outnl()
	scope = ast.NewScope(scope)
	scope.Insert(ast.NewObj(ast.Fun, d.Name.Name))
	this.genBlockStmt(scope, d.Body)
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
		this.genCaseClause(scope, t)
	case *ast.SendStmt:
		this.genSendStmt(scope, t)
	case *ast.ReturnStmt:
		this.genReturnStmt(scope, t)
	default:
		log.Println("unknown", reflect.TypeOf(stmt), t)
	}
}
func (c *g2nc) genAssignStmt(scope *ast.Scope, s *ast.AssignStmt) {
	// log.Println(s.Tok.String(), s.Tok.Precedence(), s.Tok.IsOperator(), s.Tok.IsLiteral(), s.Lhs)
	for i := 0; i < len(s.Rhs); i++ {
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
				c.out(c.chanElemTypeName(chexpr))
				c.genExpr(scope, s.Lhs[i])
				c.outfh().outnl()
			}

			var ns = putscope(scope, ast.Var, "varname", s.Lhs[i])
			c.genExpr(ns, s.Rhs[i])
		} else if isidxas {
			if s.Tok.String() == ":=" {
				c.out(c.exprTypeName(scope, s.Rhs[i]))
			}
			var ns = putscope(scope, ast.Var, "varval", s.Rhs[i])
			c.genExpr(ns, s.Lhs[i])
		} else {
			if s.Tok.String() == ":=" {
				c.out(c.exprTypeName(scope, s.Rhs[i]))
			}
			c.genExpr(scope, s.Lhs[i])

			var ns = putscope(scope, ast.Var, "varname", s.Lhs[i])
			c.out(" = ")
			c.genExpr(ns, s.Rhs[i])
			c.outfh().outnl()
		}
	}

}
func (this *g2nc) genGoStmt(scope *ast.Scope, stmt *ast.GoStmt) {
	// calleename := stmt.Call.Fun.(*ast.Ident).Name
	// this.genCallExpr(scope, stmt.Call)
	// define function in function in c?
	// this.genRoutineStargs(scope, stmt.Call)
	// this.genRoutineStwrap(scope, stmt.Call)
	this.genRoutineWcall(scope, stmt.Call)
}
func (c *g2nc) genRoutineStargs(scope *ast.Scope, e *ast.CallExpr) {
	funame := e.Fun.(*ast.Ident).Name
	if _, ok := c.psctx.grstargs[funame]; ok {
		return
	}

	c.out("typedef struct {")
	for idx, ae := range e.Args {
		fldname := fmt.Sprintf("a%d", idx)
		fldtype := c.exprTypeName(scope, ae)
		log.Println(funame, fldtype, fldname, reflect.TypeOf(ae))
		c.out(fldtype, fldname).outfh().outnl()
	}
	c.out("}", funame+"_routine_args").outfh().outnl()
}
func (c *g2nc) genRoutineStwrap(scope *ast.Scope, e *ast.CallExpr) {
	funame := e.Fun.(*ast.Ident).Name
	if _, ok := c.psctx.grstargs[funame]; ok {
		return
	}
	c.psctx.grstargs[funame] = true

	stname := funame + "_routine_args"
	c.out("void", funame+"_routine", "(void* vpargs)").outnl()
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
func (c *g2nc) genRoutineWcall(scope *ast.Scope, e *ast.CallExpr) {
	funame := e.Fun.(*ast.Ident).Name
	wfname := funame + "_routine"
	stname := funame + "_routine_args"

	c.out("// gogorun", funame).outnl()
	c.out("{")
	c.out(stname, "*args = (", stname, "*)GC_malloc(sizeof(", stname, "))").outfh().outnl()
	for idx, arg := range e.Args {
		c.out(fmt.Sprintf("args->a%d", idx), "=")
		c.genExpr(scope, arg)
		c.outfh().outnl()
	}
	c.out(fmt.Sprintf("cxrt_routine_post(%s, args);", wfname)).outnl()
	c.out("}").outnl()
}

func (c *g2nc) genForStmt(scope *ast.Scope, s *ast.ForStmt) {
	isefor := s.Init == nil && s.Cond == nil && s.Post == nil // for {}
	c.out("for (")
	c.genStmt(scope, s.Init, 0)
	// c.out(";") // TODO ast.AssignStmt has put ;
	if isefor {
		c.out(";")
	}
	c.genExpr(scope, s.Cond)
	c.out(";")
	c.genStmt(scope, s.Post, 0)
	c.out(")")
	c.genBlockStmt(scope, s.Body)
}
func (c *g2nc) genRangeStmt(scope *ast.Scope, s *ast.RangeStmt) {
	varty := c.info.TypeOf(s.X)
	log.Println(varty, reflect.TypeOf(varty))
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
		c.out("}").outnl()
	case *types.Slice:
		keyidstr := fmt.Sprintf("%v", s.Key)
		keyidstr = gopp.IfElseStr(keyidstr == "_", "idx", keyidstr)
		valtystr := c.exprTypeName(scope, s.Value)

		c.out("{").outnl()
		c.outf("  for (int %s = 0; %s < array_size(%v); %s++) {",
			keyidstr, keyidstr, s.X, keyidstr).outnl()
		c.outf("     %s %v = {0}", valtystr, s.Value).outfh().outnl()
		var tmpvar = tmpvarname()
		c.outf("    void* %s = {0}", tmpvar).outfh().outnl()
		c.outf("    int rv = array_get_at(%v, %s, (void**)&%s)", s.X, keyidstr, tmpvar).outfh().outnl()
		c.outf("    %v = (%v)(uintptr_t)%s", s.Value, valtystr, tmpvar).outfh().outnl()
		c.genBlockStmt(scope, s.Body)
		c.out("  }").outnl()
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
		c.out("else")
		c.genStmt(scope, s.Else, 0)
	}
}
func (c *g2nc) genSwitchStmt(scope *ast.Scope, s *ast.SwitchStmt) {
	tagty := c.info.TypeOf(s.Tag)
	if tagty == nil {
		log.Println(tagty, c.exprpos(s.Tag))
	}
	log.Println(tagty, reflect.TypeOf(tagty), reflect.TypeOf(tagty.Underlying()))
	switch tty := tagty.(type) {
	case *types.Basic:
		switch tty.Kind() {
		case types.Int:
			c.genSwitchStmtNum(scope, s)
		default:
			log.Println("unknown", tagty, reflect.TypeOf(tagty))
		}
	default:
		log.Println("unknown", tagty, reflect.TypeOf(tagty))
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
func (c *g2nc) genCaseClause(scope *ast.Scope, s *ast.CaseClause) {
	log.Println(s.List, s.Body)
	if len(s.List) == 0 {
		// default
		c.out("default:").outnl()
		for idx, s_ := range s.Body {
			c.genStmt(scope, s_, idx)
		}
	} else {
		if len(s.List) != len(s.Body) {
			log.Println("wtf", s.List, s.Body)
		}
		// TODO precheck if have fallthrough
		for idx, e := range s.List {
			c.out("case")
			c.genExpr(scope, e)
			c.out(":").outnl()
			c.genStmt(scope, s.Body[idx], idx)
			c.out("break").outfh().outnl()
		}
	}
}

func (c *g2nc) genCallExpr(scope *ast.Scope, te *ast.CallExpr) {
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
		} else if funame == "delete" {
			c.genCallExprDelete(scope, te)
		} else if funame == "println" {
			c.genCallExprPrintln(scope, te)
		} else {
			c.genCallExprNorm(scope, te)
		}
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
		c.outf("cxstring_len(%s)", arg0.(*ast.Ident).Name)
	} else if isslicety(argty.String()) {
		funame := te.Fun.(*ast.Ident).Name
		if funame == "len" {
			c.outf("array_size(%s)", arg0.(*ast.Ident).Name)
		} else if funame == "cap" {
			c.outf("array_capacity(%s)", arg0.(*ast.Ident).Name)
		} else {
			panic(funame)
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
		default:
			log.Println("todo", reflect.TypeOf(arg1), arg1)
		}
		c.outf("hashtable_remove(%s, (void*)(uintptr_t)%s, 0)",
			arg0.(*ast.Ident).Name, keystr).outfh().outnl()
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
	c.out("__FILE__, __LINE__, __FUNCTION__", gopp.IfElseStr(len(te.Args) > 0, ",", ""))
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
		} else {
			c.genExpr(scope, e1)
		}
		c.out(gopp.IfElseStr(idx == len(te.Args)-1, "", ", "))
	}
	c.out(")")

	// check if real need, ;\n
	cs := c.psctx.cursors[te]
	if cs.Name() != "Args" {
		c.outfh().outnl()
	}
}
func (c *g2nc) genCallExprNorm(scope *ast.Scope, te *ast.CallExpr) {
	// funame := te.Fun.(*ast.Ident).Name
	c.genExpr(scope, te.Fun)
	c.out("(")
	for idx, e1 := range te.Args {
		c.genExpr(scope, e1)
		c.out(gopp.IfElseStr(idx == len(te.Args)-1, "", ", "))
	}
	c.out(")")

	// check if real need, ;\n
	cs := c.psctx.cursors[te]
	if cs.Name() != "Args" {
		c.outfh().outnl()
	}
}

// chan structure args
func (c *g2nc) genChanStargs(scope *ast.Scope, e ast.Expr) {
	var elemtyname = c.chanElemTypeName(e)
	// typedef struct { int  elem; } chan_arg_int;
	c.out("typedef struct {", elemtyname, " elem;} chan_arg_"+elemtyname).outfh().outnl()
}
func (c *g2nc) genSendStmt(scope *ast.Scope, s *ast.SendStmt) {
	var elemtyname = c.chanElemTypeName(s.Chan)
	var chanargname = "chan_arg_" + elemtyname
	c.out("{").outnl()
	c.outf("%s* args = (%s*)GC_malloc(sizeof(%s))", chanargname, chanargname, chanargname).outfh().outnl()
	c.out("args->elem = ")
	c.genExpr(scope, s.Value)
	c.outfh().outnl()
	c.outf("cxrt_chan_send(")
	c.genExpr(scope, s.Chan)
	c.out(", args)").outfh().outnl()
	c.out("}").outnl()
}
func (c *g2nc) genRecvStmt(scope *ast.Scope, e ast.Expr) {
	var elemtyname = c.chanElemTypeName(e)
	var chanargname = "chan_arg_" + elemtyname

	c.out("{")
	c.out("void* rvx = cxrt_chan_recv(")
	c.genExpr(scope, e)
	c.out(")").outfh().outnl()
	c.out(" // c = rv->v").outfh().outnl()
	c.outf("%s rvp = ((%s*)rvx)->elem", elemtyname, chanargname).outfh().outnl()

	varobj := scope.Lookup("varname")
	if varobj != nil {
		c.genExpr(scope, varobj.Data.(ast.Expr)) // left
		c.out("= rvp").outfh().outnl()
	}

	c.out("}").outnl()
}
func (c *g2nc) chanElemTypeName(e ast.Expr) string {
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
		default:
			log.Println("unknown", t)
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
	c.out("return")
	for idx, ae := range e.Results {
		c.genExpr(scope, ae)
		if idx < len(e.Results)-1 {
			c.out(",") // TODO
		}
	}
	c.outfh().outnl().outnl()
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
		this.genTypeExpr(scope, fld.Type)
		if withname && len(fld.Names) > 0 {
			this.genExpr(scope, fld.Names[0])
		}
		outskip := skiplast && (idx == len(flds.List)-1)
		this.out(gopp.IfElseStr(outskip, "", linebrk))
	}
}

func (this *g2nc) genTypeExpr(scope *ast.Scope, e ast.Expr) {
	switch te := e.(type) {
	case *ast.Ident:
		ety := this.info.TypeOf(e)
		if ety == nil {
			log.Println(reflect.TypeOf(e))
			panic(e)
		}
		switch be := ety.(type) {
		case *types.Basic:
			idname := gopp.IfElseStr(te.Name == "nil", "nilptr", te.Name)
			idname = gopp.IfElseStr(te.Name == "string", "cxstring*", te.Name)
			this.out(idname, " ")
			if be == nil {
			}
		default:
			this.out(te.Name)
		}
	case *ast.StarExpr:
		this.genTypeExpr(scope, te.X)
		this.out("*")
	case *ast.MapType:
		this.outf("/*%v=>%v*/HashTable*", te.Key, te.Value)
	case *ast.ArrayType:
		this.outf("/*%v*/Array*", te.Elt)
	default:
		log.Println(e, reflect.TypeOf(e))
	}
}

func (this *g2nc) genExpr(scope *ast.Scope, e ast.Expr) {
	// log.Println(reflect.TypeOf(e))
	switch te := e.(type) {
	case *ast.Ident:
		// log.Println(te.Name, te.String(), te.IsExported(), te.Obj, reflect.TypeOf(e))
		idname := te.Name
		idname = gopp.IfElseStr(idname == "nil", "nilptr", idname)
		idname = gopp.IfElseStr(idname == "string", "cxstring*", idname)
		this.out(idname, " ")
	case *ast.ArrayType:
		log.Println("unimplemented", te, reflect.TypeOf(e), e.Pos())
	case *ast.StructType:
		this.genFieldList(scope, te.Fields, false, true, ";\n", false)
	case *ast.UnaryExpr:
		log.Println(te.Op.String(), te.X)
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
			this.out(fmt.Sprintf("(%v*)GC_malloc(sizeof(%v));", t2.Type, t2.Type)).outnl()
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
				default:
					log.Println("unknown", idx, reflect.TypeOf(ex))
				}
			}
		case *ast.ArrayType:
			this.outf("cxarray_new()").outfh().outnl()
			var vo = scope.Lookup("varname")
			for idx, ex := range te.Elts {
				this.genCxarrAdd(scope, vo.Data, ex, idx)
			}
			if be == nil {
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
			case types.Int, types.UntypedInt:
				this.out(te.Value)
			case types.String, types.UntypedString:
				this.out(fmt.Sprintf("cxstring_new_cstr(%s)", te.Value))
			case types.Float64, types.Float32:
				this.out(te.Value)
			case types.Uint8, types.Int8, types.Uint32, types.Int32:
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
			this.genCxmapAddkv(scope, te.X, te.Index, vo.Data.(ast.Expr))
		} else if isslicety(varty.String()) {
			// get or set?
			if vo == nil { // right value
				this.genCxarrGet(scope, te.X, te.Index)
			} else { // left value
				this.genCxarrSet(scope, te.X, te.Index, vo.Data)
			}
		} else {
			log.Println("todo", te.X, te.Index)
			log.Println("todo", reflect.TypeOf(te.X))
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
		this.genExpr(scope, te.X)
		this.out("->")
		this.genExpr(scope, te.Sel)
	case *ast.StarExpr:
		this.genExpr(scope, te.X)
		this.out("*")
	default:
		log.Println("unknown", reflect.TypeOf(e), e, te)
	}
}
func (c *g2nc) genCxmapAddkv(scope *ast.Scope, vnamex interface{}, ke, ve ast.Expr) {
	keystr := ""
	switch be := ke.(type) {
	case *ast.BasicLit:
		switch be.Kind {
		case token.STRING:
			// keystr = fmt.Sprintf("cxhashtable_hash_str(%s)", be.Value)
			keystr = fmt.Sprintf("cxstring_new_cstr(%s)", be.Value)
		default:
			log.Println("unknown index key kind", be.Kind)
		}
	case *ast.Ident:
		varty := c.info.TypeOf(ke)
		switch varty.String() {
		case "string":
			// keystr = fmt.Sprintf("cxhashtable_hash_str2(%s->ptr, %s->len)", be.Name, be.Name)
			keystr = be.Name
		default:
			log.Println("unknown", varty, ke)
		}
	case *ast.SelectorExpr:
		varty := c.info.TypeOf(ke)
		switch varty.String() {
		case "string":
			sym := fmt.Sprintf("%v->%v", be.X, be.Sel)
			// keystr = fmt.Sprintf("cxhashtable_hash_str2((%s)->ptr, (%s)->len)", sym, sym)
			keystr = sym
		default:
			log.Println("unknown", varty, ke)
		}
	default:
		log.Println("unknown index key", ke, reflect.TypeOf(ke))
	}

	valstr := ""
	switch be := ve.(type) {
	case *ast.BasicLit:
		valstr = be.Value
	case *ast.Ident:
		valstr = be.Name
	default:
		log.Println("unknown", ve, reflect.TypeOf(ke), reflect.TypeOf(ve))
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
		varstr, keystr, valstr).outfh().outnl()
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
	c.outf("array_add(%v, (void*)(uintptr_t)%v)", vnamex, valstr).outfh().outnl()
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
	switch te := e.(type) {
	case *ast.Ident:
		ety := this.info.TypeOf(e)
		switch rety := ety.(type) {
		case *types.Chan:
			return "void*"
		case *types.Basic:
			switch rety.Kind() {
			case types.Bool, types.UntypedBool:
				return "bool"
			case types.String:
				return "cxstring*"
			case types.Int:
				return "int"
			case types.Uint:
				return "uint"
			case types.Rune:
				return "rune"
			case types.Int8:
				return "int8"
			case types.Uint8:
				return "uint8"
			default:
				log.Println("todo", rety)
			}
		case *types.Named:
			return te.Name
		case *types.Pointer:
			ety := this.info.TypeOf(e)
			return sign2rety(ety.String())
		default:
			log.Println("unknown", ety, rety, reflect.TypeOf(ety))
		}
		return te.Name
	case *ast.ArrayType:
		return "Array*"
	case *ast.StructType:
		log.Println("todo")
	case *ast.UnaryExpr:
		switch te.Op {
		case token.AND:
			return this.exprTypeName(scope, te.X) + "*"
		case token.NOT:
			return "bool"
		default:
			log.Println("todo", te)
		}
	case *ast.CompositeLit:
		return this.exprTypeName(scope, te.Type)
	case *ast.BasicLit:
		tyname := strings.ToLower(te.Kind.String())
		if tyname == "string" {
			tyname = "cxstring*"
		}
		return tyname
	case *ast.CallExpr:
		switch be := te.Fun.(type) {
		case *ast.Ident:
			switch te.Fun.(*ast.Ident).Name {
			case "make":
				return this.exprTypeName(scope, te.Args[0])
			case "len", "cap":
				return "int"
			default:
				rety := this.info.TypeOf(te.Fun)
				// log.Println(rety, reflect.TypeOf(rety))
				// log.Println(rety.Underlying(), reflect.TypeOf(rety.Underlying()))
				return sign2rety(rety.String())
			}
		default:
			log.Println("todo", be, reflect.TypeOf(be))
		}
	case *ast.ChanType:
		return "void*"
	case *ast.MapType:
		return "HashTable*"
	case *ast.SliceExpr:
		varty := this.info.TypeOf(te.X)
		if isstrty2(varty) {
			return "cxstring*"
		} else if isslicety2(varty) {
			return "Array*"
		} else {
			log.Println("toto", varty)
		}
	case *ast.IndexExpr:
		varty := this.info.TypeOf(e)
		return strings.ToLower(varty.String())
	case *ast.BinaryExpr:
		if te.Op.IsOperator() {
			switch te.Op {
			case token.EQL, token.NEQ:
				return "bool"
			default:
				log.Println("todo", te.Op)
			}
		} else {
			log.Println("todo", te)
		}
	default:
		log.Println("unknown", reflect.TypeOf(e), te, this.info.TypeOf(e))
	}
	return "unknownty"
}
func (this *g2nc) exprTypeFmt(scope *ast.Scope, e ast.Expr) string {

	ety := this.info.TypeOf(e)
	// log.Println(reflect.TypeOf(e), ety)
	if ety == nil {
		switch t := e.(type) {
		case *ast.CallExpr:
			// TODO builtin type preput to types.Info
			switch t.Fun.(*ast.Ident).Name {
			case "gettid":
				return "d"
			default:
				log.Println("unknown", t)
			}
		case *ast.Ident:
			return "d"
		default:
			log.Println("unknown", e, reflect.TypeOf(e))
		}
		return ""
	}

	// eval := this.info.Types[e]
	switch t := ety.Underlying().(type) {
	case *types.Pointer:
		return "p"
	case *types.Basic:
		switch t.Kind() {
		case types.Int:
			return "d"
		case types.String:
			return ".*s"
		case types.Bool:
			return "d"
		case types.Rune:
			return "d"
		case types.Invalid:
			return "d" // TODO
		default:
			log.Println("unknown", t, e, ety)
		}
	case *types.Map:
		// log.Println(t.String(), t.Key(), t.Elem())
		return "p"
	case *types.Slice:
		return "p"
	default:
		log.Println("unknown", t, reflect.TypeOf(ety))
	}
	return ""
}

func (this *g2nc) genGenDecl(scope *ast.Scope, d *ast.GenDecl) {
	log.Println(d.Tok, d.Specs, len(d.Specs), d.Tok.IsKeyword(), d.Tok.IsLiteral(), d.Tok.IsOperator())
	for _, spec := range d.Specs {
		switch tspec := spec.(type) {
		case *ast.TypeSpec:
			this.genTypeSpec(scope, tspec)
		case *ast.ValueSpec:
			this.genValueSpec(scope, tspec)
		case *ast.ImportSpec:
			log.Println("todo", reflect.TypeOf(d), reflect.TypeOf(spec), tspec.Path)
		default:
			log.Println("unknown", reflect.TypeOf(d), reflect.TypeOf(spec))
		}
	}
}
func (this *g2nc) genTypeSpec(scope *ast.Scope, spec *ast.TypeSpec) {
	// log.Println(spec.Name, spec.Type)
	this.outf("struct %s {", spec.Name.Name)
	this.outnl()
	this.genExpr(scope, spec.Type)
	this.out("}").outfh().outnl()
	this.outf("typedef struct %s %s", spec.Name.Name, spec.Name.Name).outfh()
	this.outnl().outnl()
}
func putscope(scope *ast.Scope, k ast.ObjKind, name string, value interface{}) *ast.Scope {
	var pscope = ast.NewScope(scope)
	var varobj = ast.NewObj(k, name)
	varobj.Data = value
	pscope.Insert(varobj)
	return pscope
}
func (c *g2nc) genValueSpec(scope *ast.Scope, spec *ast.ValueSpec) {
	for idx, name := range spec.Names {
		if spec.Type == nil {
			if len(spec.Names) != len(spec.Values) {
				log.Println("todo", idx, len(spec.Names), len(spec.Values), spec.Names, spec.Values)
			}
			v := spec.Values[idx]
			c.out(c.exprTypeName(scope, v))
		} else {
			c.genExpr(scope, spec.Type)
		}
		c.out(name.Name)
		if idx < len(spec.Values) {
			var ns = putscope(scope, ast.Var, "varname", name)
			c.out("=")
			c.genExpr(ns, spec.Values[idx])
		} else {
			// TODO set default value
			c.out("=", "{0}")
		}
		c.outfh().outnl()
	}
}

func (this *g2nc) outeq() *g2nc   { return this.out("=") }
func (this *g2nc) outstar() *g2nc { return this.out("*") }
func (this *g2nc) outfh() *g2nc   { return this.out(";") }
func (this *g2nc) outnl() *g2nc   { return this.out("\n") }
func (this *g2nc) out(ss ...string) *g2nc {
	for _, s := range ss {
		// fmt.Print(s, " ")
		this.sb.WriteString(s + " ")
	}
	return this
}
func (this *g2nc) outf(format string, args ...interface{}) *g2nc {
	s := fmt.Sprintf(format, args...)
	this.out(s)
	return this
}

func (this *g2nc) code() (string, string) {
	code := "#include <cxrtbase.h>\n\n"
	code += this.sb.String()
	return code, "c"
}
