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

type g2nc struct {
	psctx *ParserContext
	sb    strings.Builder

	info *types.Info
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
	log.Println(name)
	for name, f := range pkg.Files {
		this.genfile(pkg.Scope, name, f)
	}
}
func (this *g2nc) genfile(scope *ast.Scope, name string, f *ast.File) {
	log.Println(name)

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
	astutil.Apply(d, func(c *astutil.Cursor) bool {
		switch t := c.Node().(type) {
		case *ast.GoStmt:
			gostmts = append(gostmts, t)
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
		this.out("{ // func init() }").outnl()
	}
	scope = ast.NewScope(scope)
	for idx, s := range stmt.List {
		this.genStmt(scope, s, idx)
	}
	this.out("}").outnl()
}

func (this *g2nc) genStmt(scope *ast.Scope, stmt ast.Stmt, idx int) {
	switch t := stmt.(type) {
	case *ast.AssignStmt:
		log.Println(t.Tok.String(), t.Lhs)
		for i := 0; i < len(t.Rhs); i++ {
			this.out(this.exprTypeName(scope, t.Rhs[i]))
			this.genExpr(scope, t.Lhs[i])
			this.out(" = ")
			this.genExpr(scope, t.Rhs[i])
		}
	case *ast.ExprStmt:
		this.genExpr(scope, t.X)
	case *ast.GoStmt:
		this.genGoStmt(scope, t)
	default:
		log.Println(reflect.TypeOf(stmt), t)
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
		log.Println(funame, fldtype, fldname)
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

func (c *g2nc) genCallExpr(scope *ast.Scope, e *ast.CallExpr) {
	c.genExpr(scope, e)
}

// keepvoid
// skiplast 作用于linebrk
func (this *g2nc) genFieldList(scope *ast.Scope, flds *ast.FieldList,
	keepvoid bool, withname bool, linebrk string, skiplast bool) {
	log.Println(flds, keepvoid)
	if keepvoid && (flds == nil || flds.NumFields() == 0) {
		this.out("void")
		return
	}
	if flds == nil {
		return
	}

	for idx, fld := range flds.List {
		_, _ = idx, fld
		this.genExpr(scope, fld.Type)
		if withname && len(fld.Names) > 0 {
			this.genExpr(scope, fld.Names[0])
		}
		outskip := skiplast && (idx == len(flds.List)-1)
		this.out(gopp.IfElseStr(outskip, "", linebrk))
	}
}

func (this *g2nc) genExpr(scope *ast.Scope, e ast.Expr) {
	// log.Println(reflect.TypeOf(e))
	switch te := e.(type) {
	case *ast.Ident:
		// log.Println(te.Name, te.String(), te.IsExported(), te.Obj)
		this.out(te.Name, " ")
	case *ast.ArrayType:
		log.Println("unimplemented", te, reflect.TypeOf(e))
	case *ast.StructType:
		this.genFieldList(scope, te.Fields, false, true, ";\n", false)
	case *ast.UnaryExpr:
		log.Println(te.Op.String(), te.X)
		switch t2 := te.X.(type) {
		case *ast.CompositeLit:
			this.out(fmt.Sprintf("(%v*)GC_malloc(sizeof(%v));", t2.Type, t2.Type)).outnl()
		default:
			log.Println(reflect.TypeOf(te), t2)
		}
		this.genExpr(scope, te.X)
	case *ast.CompositeLit:
		log.Println(te.Type, te.Elts)
	case *ast.CallExpr:
		funame := te.Fun.(*ast.Ident).Name
		this.genExpr(scope, te.Fun)
		this.out("(")
		if funame == "println" && len(te.Args) > 0 {
			var tyfmts []string
			for _, e := range te.Args {
				tyfmt := this.exprTypeFmt(scope, e)
				tyfmts = append(tyfmts, "%"+tyfmt)
			}
			this.out(fmt.Sprintf(`"%s"`, strings.Join(tyfmts, " ")))
			this.out(", ")
		}
		for idx, e := range te.Args {
			this.genExpr(scope, e)
			this.out(gopp.IfElseStr(idx == len(te.Args)-1, "", ", "))
		}
		this.out(")")

		// check if real need, ;\n
		c := this.psctx.cursors[te]
		if c.Name() != "Args" {
			this.outfh().outnl()
		}
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
	default:
		log.Println("unknown", reflect.TypeOf(e), te)
	}
}
func (this *g2nc) exprTypeName(scope *ast.Scope, e ast.Expr) string {
	switch te := e.(type) {
	case *ast.Ident:
		return te.Name
	case *ast.ArrayType:
	case *ast.StructType:
	case *ast.UnaryExpr:
		return this.exprTypeName(scope, te.X) + "*"
	case *ast.CompositeLit:
		return this.exprTypeName(scope, te.Type)
	case *ast.BasicLit:
		return strings.ToLower(te.Kind.String())
	default:
		log.Println(reflect.TypeOf(e), te)
	}
	return ""
}
func (this *g2nc) exprTypeFmt(scope *ast.Scope, e ast.Expr) string {

	ety := this.info.TypeOf(e)
	log.Println(reflect.TypeOf(e), ety)
	if ety == nil {
		switch t := e.(type) {
		case *ast.CallExpr:
			// TODO builtin type preput to types.Info
			switch t.Fun.(*ast.Ident).Name {
			case "gettid":
				return "d"
			}
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

func (this *g2nc) genGenDecl(scope *ast.Scope, d *ast.GenDecl) {
	log.Println(d.Tok, d.Specs, len(d.Specs), d.Tok.IsKeyword(), d.Tok.IsLiteral(), d.Tok.IsOperator())
	log.Println(reflect.TypeOf(d.Specs))
	for _, spec := range d.Specs {
		log.Println(reflect.TypeOf(spec))
		switch tspec := spec.(type) {
		case *ast.TypeSpec:
			this.genTypeSpec(scope, tspec)
		default:
			log.Println(reflect.TypeOf(d))
		}
	}
}
func (this *g2nc) genTypeSpec(scope *ast.Scope, spec *ast.TypeSpec) {
	log.Println(spec.Name, spec.Type)
	this.out("typedef struct {")
	this.outnl()
	this.genExpr(scope, spec.Type)
	this.out("}", spec.Name.Name, ";")
	this.outnl()
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

func (this *g2nc) code() string {
	return this.sb.String()
}
