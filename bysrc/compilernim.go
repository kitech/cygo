package main

import (
	"fmt"
	"go/ast"
	"go/types"
	"gopp"
	"log"
	"reflect"
	"strings"

	"golang.org/x/tools/go/ast/astutil"
)

type g2nim struct {
	psctx  *ParserContext
	sb     strings.Builder
	indent int // keep nim indent carefully

	info *types.Info
}

func (this *g2nim) genpkgs() {
	this.info = &this.psctx.info

	// pkgs order?
	for pname, pkg := range this.psctx.pkgs {
		pkg.Scope = ast.NewScope(nil)
		this.genpkg(pname, pkg)
	}
}

func (this *g2nim) genpkg(name string, pkg *ast.Package) {
	log.Println(name)
	for name, f := range pkg.Files {
		this.genfile(pkg.Scope, name, f)
	}
}
func (this *g2nim) genfile(scope *ast.Scope, name string, f *ast.File) {
	log.Println(name)

	// decls order?
	for _, d := range f.Decls {
		this.genDecl(scope, d)
	}
}

func (this *g2nim) genDecl(scope *ast.Scope, d ast.Decl) {
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
func (this *g2nim) genPreFuncDecl(scope *ast.Scope, d *ast.FuncDecl) {
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
	return
	for _, stmt := range gostmts {
		this.genRoutineStargs(scope, stmt.Call)
		this.genRoutineStwrap(scope, stmt.Call)
	}
	for _, stmt := range sdstmts {
		this.genChanStargs(scope, stmt.Chan) // chan structure args
	}
}
func (this *g2nim) genFuncDecl(scope *ast.Scope, d *ast.FuncDecl) {
	this.out("proc ", d.Name.String())
	this.out("(")
	this.genFieldList(scope, d.Type.Params, false, true, ",", true)
	this.out(")")
	this.outmhsp()
	this.genFieldList(scope, d.Type.Results, true, false, "", false)
	this.outspeq().outnl()
	scope = ast.NewScope(scope)
	scope.Insert(ast.NewObj(ast.Fun, d.Name.Name))
	this.genBlockStmt(scope, d.Body)

	this.incrindent()
	this.outindent()
	this.out("# end")
	this.decrindent()
	this.outnl().outnl()
}

func (this *g2nim) genBlockStmt(scope *ast.Scope, stmt *ast.BlockStmt) {
	this.incrindent()
	if scope.Lookup("main") != nil {
		// this.out("{ cxrt_init_env(); }").outnl()
		// this.out("{ \n // func init() \n }").outnl()
	}
	scope = ast.NewScope(scope)
	for idx, s := range stmt.List {
		this.outindent()
		this.genStmt(scope, s, idx)
	}
	this.decrindent()
}

// clause index?
func (this *g2nim) genStmt(scope *ast.Scope, stmt ast.Stmt, idx int) {
	switch t := stmt.(type) {
	case *ast.AssignStmt:
		this.genAssignStmt(scope, t)
	case *ast.ExprStmt:
		this.genExpr(scope, t.X)
	case *ast.GoStmt:
		this.genGoStmt(scope, t)
	case *ast.ForStmt:
		this.genForStmt(scope, t)
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

// data: ast.Expr or some else
func (c *g2nim) addScopeData(scope *ast.Scope, name string, data interface{}) *ast.Scope {
	var nscope = ast.NewScope(scope)
	var varobj = ast.NewObj(ast.Var, name)
	varobj.Data = data
	nscope.Insert(varobj)
	return nscope
}
func (c *g2nim) genAssignStmt(scope *ast.Scope, s *ast.AssignStmt) {
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
		}

		var scope = ast.NewScope(scope)
		var varobj = ast.NewObj(ast.Var, "varname")
		varobj.Data = s.Lhs[i]
		scope.Insert(varobj)

		if ischrv {
			if s.Tok.String() == ":=" {
				c.out(c.chanElemTypeName(chexpr))
				c.genExpr(scope, s.Lhs[i])
				c.outfh().outnl()
			}

			c.genExpr(scope, s.Rhs[i])

		} else {
			if s.Tok.String() == ":=" {
				c.out("var ")
			}
			c.genExpr(scope, s.Lhs[i])

			c.out(" = ")
			c.genExpr(scope, s.Rhs[i])
		}
	}

}
func (this *g2nim) genGoStmt(scope *ast.Scope, stmt *ast.GoStmt) {
	// calleename := stmt.Call.Fun.(*ast.Ident).Name
	this.out("spawn").outsp()
	this.genCallExpr(scope, stmt.Call)
	// define function in function in c?
	// this.genRoutineStargs(scope, stmt.Call)
	// this.genRoutineStwrap(scope, stmt.Call)
	// this.genRoutineWcall(scope, stmt.Call)
}
func (c *g2nim) genRoutineStargs(scope *ast.Scope, e *ast.CallExpr) {
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
func (c *g2nim) genRoutineStwrap(scope *ast.Scope, e *ast.CallExpr) {
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
func (c *g2nim) genRoutineWcall(scope *ast.Scope, e *ast.CallExpr) {
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

func (c *g2nim) genForStmt(scope *ast.Scope, s *ast.ForStmt) {
	c.out("for (")
	c.genStmt(scope, s.Init, 0)
	c.out(";")
	c.genExpr(scope, s.Cond)
	c.out(";")
	c.genStmt(scope, s.Post, 0)
	c.out(")")
	c.genBlockStmt(scope, s.Body)
}
func (c *g2nim) genIncDecStmt(scope *ast.Scope, s *ast.IncDecStmt) {
	c.genExpr(scope, s.X)
	if s.Tok.IsOperator() {
		c.out(s.Tok.String())
	}
}
func (c *g2nim) genBranchStmt(scope *ast.Scope, s *ast.BranchStmt) {
	c.out(s.Tok.String())
	if s.Label != nil {
		c.out(s.Label.Name)
	}
	c.outfh().outnl()
}
func (c *g2nim) genDeclStmt(scope *ast.Scope, s *ast.DeclStmt) {
	c.genDecl(scope, s.Decl)
}

func (c *g2nim) genIfStmt(scope *ast.Scope, s *ast.IfStmt) {
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
func (c *g2nim) genSwitchStmt(scope *ast.Scope, s *ast.SwitchStmt) {
	tagty := c.info.TypeOf(s.Tag)
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
func (c *g2nim) genSwitchStmtNum(scope *ast.Scope, s *ast.SwitchStmt) {
	c.out("switch (")
	c.genExpr(scope, s.Tag)
	c.out(")")
	c.genBlockStmt(scope, s.Body)
}
func (c *g2nim) genSwitchStmtStr(scope *ast.Scope, s *ast.SwitchStmt) {
	log.Println(s.Tag)
}
func (c *g2nim) genCaseClause(scope *ast.Scope, s *ast.CaseClause) {
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

func (c *g2nim) genCallExpr(scope *ast.Scope, te *ast.CallExpr) {
	funame := te.Fun.(*ast.Ident).Name
	if funame == "make" {
		c.genCallExprMake(scope, te)
	} else {
		c.genCallExprNorm(scope, te)
	}
}
func (c *g2nim) genCallExprMake(scope *ast.Scope, te *ast.CallExpr) {
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
func (c *g2nim) genCallExprNorm(scope *ast.Scope, te *ast.CallExpr) {
	// c.genExpr(scope, e)
	// check if need a discard return value
	// 有返回值，并且没有赋值给变量，并且不在函数调用列表，并且没有前置 spawn语句
	goty := c.info.TypeOf(te)
	if goty != nil {
		hasret := true
		switch t := goty.(type) {
		case *types.Tuple:
			hasret = t.Len() > 0
		case *types.Basic:
			hasret = true
		default:
			hasret = false
			log.Println(goty.String(), te.Fun, reflect.TypeOf(goty))
		}

		if hasret {
			obj0 := scope.Lookup("valspec")
			if obj0 == nil {
				c.out("discard").outsp()
			}
		}
	}

	funame := te.Fun.(*ast.Ident).Name
	c.genExpr(scope, te.Fun)
	c.out("(")
	if funame == "println" && len(te.Args) > 0 {
		var tyfmts []string
		for _, e1 := range te.Args {
			tyfmt := c.exprTypeFmt(scope, e1)
			tyfmts = append(tyfmts, "%"+tyfmt)
		}
		c.out(fmt.Sprintf(`"%s"`, strings.Join(tyfmts, " ")))
		c.out(", ")
	}
	for idx, e1 := range te.Args {
		c.genExpr(scope, e1)
		c.out(gopp.IfElseStr(idx == len(te.Args)-1, "", ", "))
	}
	c.out(")")

	// check if real need, ;\n
	cs := c.psctx.cursors[te]
	if cs.Name() != "Args" {
		c.outnl()
	}
}

// chan structure args
func (c *g2nim) genChanStargs(scope *ast.Scope, e ast.Expr) {
	var elemtyname = c.chanElemTypeName(e)
	// typedef struct { int  elem; } chan_arg_int;
	c.out("typedef struct {", elemtyname, " elem;} chan_arg_"+elemtyname).outfh().outnl()
}
func (c *g2nim) genSendStmt(scope *ast.Scope, s *ast.SendStmt) {
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
func (c *g2nim) genRecvStmt(scope *ast.Scope, e ast.Expr) {
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
func (c *g2nim) genReturnStmt(scope *ast.Scope, s *ast.ReturnStmt) {
	c.out("return ")
	c.genExpr(scope, s.Results[0])
	c.outnl()
}

func (c *g2nim) chanElemTypeName(e ast.Expr) string {
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

// keepvoid
// skiplast 作用于linebrk
func (this *g2nim) genFieldList(scope *ast.Scope, flds *ast.FieldList,
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
		if linebrk != "" {
			this.outindent()
		}

		if withname && len(fld.Names) > 0 {
			this.genExpr(scope, fld.Names[0])
			this.outmhsp()
		}
		this.genExpr(scope, fld.Type)
		outskip := skiplast && (idx == len(flds.List)-1)
		this.out(gopp.IfElseStr(outskip, "", linebrk))
	}
}

func (this *g2nim) genExpr(scope *ast.Scope, e ast.Expr) {
	// log.Println(reflect.TypeOf(e))
	switch te := e.(type) {
	case *ast.Ident:
		// log.Println(te.Name, te.String(), te.IsExported(), te.Obj)
		this.out(te.Name, "")
	case *ast.ArrayType:
		log.Println("unimplemented", te, reflect.TypeOf(e))
	case *ast.StructType:
		this.genFieldList(scope, te.Fields, false, true, "\n", false)
	case *ast.UnaryExpr:
		log.Println(te.Op.String(), te.X)
		switch te.Op.String() {
		case "<-":
			this.genRecvStmt(scope, te.X)
			return
		default:
			log.Println("unknown", te.Op.String())
		}
		switch t2 := te.X.(type) {
		case *ast.CompositeLit:
			this.out(fmt.Sprintf("new(%v)", t2.Type)).outnl()
		case *ast.UnaryExpr:
			log.Println(t2, t2.X, t2.Op)
		default:
			log.Println(reflect.TypeOf(te), t2, reflect.TypeOf(te.X))
		}
		this.genExpr(scope, te.X)
	case *ast.CompositeLit:
		log.Println(te.Type, te.Elts)
	case *ast.CallExpr:
		this.genCallExpr(scope, te)
	case *ast.BasicLit:
		ety := this.info.TypeOf(e)
		switch t := ety.Underlying().(type) {
		case *types.Basic:
			switch t.Kind() {
			case types.Int, types.UntypedInt:
				this.out(te.Value)
			case types.String, types.UntypedString:
				this.out(fmt.Sprintf("%s", te.Value))
			default:
				log.Println("unknown", t.String())
			}
		default:
			log.Println("unknown", t, reflect.TypeOf(t))
		}
	case *ast.BinaryExpr:
		this.genExpr(scope, te.X)
		this.out(te.Op.String())
		this.genExpr(scope, te.Y)
	case *ast.ChanType:
		this.out("void*")
	default:
		log.Println("unknown", reflect.TypeOf(e), e, te)
	}
}
func (this *g2nim) exprTypeName(scope *ast.Scope, e ast.Expr) string {
	switch te := e.(type) {
	case *ast.Ident:
		ety := this.info.TypeOf(e)
		switch rety := ety.(type) {
		case *types.Chan:
			return "void*"
		default:
			log.Println("unknown", ety, rety)
		}
		return te.Name
	case *ast.ArrayType:
		log.Println("todo")
	case *ast.StructType:
		log.Println("todo")
	case *ast.UnaryExpr:
		return this.exprTypeName(scope, te.X) + "*"
	case *ast.CompositeLit:
		return this.exprTypeName(scope, te.Type)
	case *ast.BasicLit:
		return strings.ToLower(te.Kind.String())
	case *ast.CallExpr:
		switch te.Fun.(*ast.Ident).Name {
		case "make":
			return this.exprTypeName(scope, te.Args[0])
		default:
			rety := this.info.TypeOf(e)
			log.Println(rety, reflect.TypeOf(rety))
		}
	case *ast.ChanType:
		return "void*"
	default:
		log.Println("unknown", reflect.TypeOf(e), te, this.info.TypeOf(e))
	}
	return "unknown"
}
func (this *g2nim) exprTypeFmt(scope *ast.Scope, e ast.Expr) string {

	ety := this.info.TypeOf(e)
	log.Println(reflect.TypeOf(e), ety)
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
		default:
			log.Println("unknown", e)
		}
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
			return "s"
		default:
			log.Println("unknown")
		}
	default:
		log.Println("unknown")
	}
	return ""
}

func (this *g2nim) genGenDecl(scope *ast.Scope, d *ast.GenDecl) {
	log.Println(d.Tok, d.Specs, len(d.Specs), d.Tok.IsKeyword(), d.Tok.IsLiteral(), d.Tok.IsOperator())
	for _, spec := range d.Specs {
		switch tspec := spec.(type) {
		case *ast.TypeSpec:
			this.genTypeSpec(scope, tspec)
		case *ast.ValueSpec:
			if len(tspec.Names) > 0 {
				scope = this.addScopeData(scope, "valspec", spec)
			}
			this.genValueSpec(scope, tspec)
		default:
			log.Println("unknown", reflect.TypeOf(d), reflect.TypeOf(spec))
		}
	}
}
func (this *g2nim) genTypeSpec(scope *ast.Scope, spec *ast.TypeSpec) {
	// log.Println(spec.Name, spec.Type)
	this.out("type") // typedef
	this.incrindent()
	this.outnl().outindent()
	this.out("P", spec.Name.Name, " = ptr ", spec.Name.Name)
	this.outnl().outindent()
	this.out(spec.Name.Name, " = ref object")
	this.incrindent()
	this.outnl()
	this.genExpr(scope, spec.Type)
	this.decrindent()
	this.decrindent()
	this.outnl()
	this.out("proc `$`(v: ", spec.Name.Name, ") : string = repr(v)").outnl()
	this.outnl().outnl()
}
func (c *g2nim) genValueSpec(scope *ast.Scope, spec *ast.ValueSpec) {
	for idx, name := range spec.Names {
		if spec.Type == nil {
			v := spec.Values[idx]
			c.out(c.exprTypeName(scope, v))
		} else {
			c.genExpr(scope, spec.Type)
		}
		c.out(name.Name)
		if idx < len(spec.Values) {
			c.out("=")
			c.genExpr(scope, spec.Values[idx])
		} else {
			// TODO set default value
			c.out("=", "{0}")
		}
		c.outfh().outnl()
	}
}

func (this *g2nim) outsp() *g2nim   { return this.out(" ") }
func (this *g2nim) outeq() *g2nim   { return this.out("=") }
func (this *g2nim) outstar() *g2nim { return this.out("*") }
func (this *g2nim) outfh() *g2nim   { return this.out(";") }
func (this *g2nim) outmhsp() *g2nim { return this.out(": ") }
func (this *g2nim) outspeq() *g2nim { return this.out(" =") }
func (this *g2nim) outnl() *g2nim   { return this.out("\n") }
func (this *g2nim) out(ss ...string) *g2nim {
	for _, s := range ss {
		// fmt.Print(s, " ")
		this.sb.WriteString(s + "")
	}
	return this
}
func (this *g2nim) outf(format string, args ...interface{}) *g2nim {
	s := fmt.Sprintf(format, args...)
	this.out(s)
	return this
}
func (this *g2nim) outindent() { this.out(strings.Repeat(" ", this.indent*4)) }
func (this *g2nim) incrindent() {
	this.indent++
}
func (this *g2nim) decrindent() {
	this.indent--
}

func (this *g2nim) code() string {
	return this.sb.String()
}
