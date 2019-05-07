package main

import (
	"fmt"
	"go/ast"
	"go/types"
	"log"
	"reflect"
	"strings"
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
		this.genFuncDecl(scope, td)
	case *ast.GenDecl:
		this.genGenDecl(scope, td)
	default:
		log.Println("unimplemented", reflect.TypeOf(d))
	}
}

func (this *g2nc) genFuncDecl(scope *ast.Scope, d *ast.FuncDecl) {
	log.Println(d.Name)
	this.genFieldList(scope, d.Type.Results, true, false, "")
	this.out(d.Name.String())
	this.out("()").outnl()
	scope = ast.NewScope(scope)
	scope.Insert(ast.NewObj(ast.Fun, d.Name.Name))
	this.genBlockStmt(scope, d.Body)
	this.outnl()
}

func (this *g2nc) genBlockStmt(scope *ast.Scope, stmt *ast.BlockStmt) {
	this.out("{").outnl()
	if scope.Lookup("main") != nil {
		this.out("{ cxrt_init_routine_env(); }").outnl()
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
			this.out(this.exprType(scope, t.Rhs[i]))
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
	calleename := stmt.Call.Fun.(*ast.Ident).Name
	this.out("// gogorun", calleename).outnl()
	this.out(fmt.Sprintf("cxrt_routine_post(%s);", calleename)).outnl()
	// this.genCallExpr(scope, stmt.Call)
}
func (c *g2nc) genCallExpr(scope *ast.Scope, e *ast.CallExpr) {
	c.genExpr(scope, e)
}

func (this *g2nc) genFieldList(scope *ast.Scope, flds *ast.FieldList, ovoid bool, withname bool, linebrk string) {
	log.Println(flds, ovoid)
	if flds == nil {
		return
	}
	if flds.NumFields() == 0 {
		this.out("void")
		return
	}

	for idx, fld := range flds.List {
		_, _ = idx, fld
		this.genExpr(scope, fld.Type)
		if withname && len(fld.Names) > 0 {
			this.genExpr(scope, fld.Names[0])
		}
		this.out(linebrk)
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
		this.genFieldList(scope, te.Fields, false, true, ";\n")
	case *ast.UnaryExpr:
		log.Println(te.Op.String(), te.X)
		switch t2 := te.X.(type) {
		case *ast.CompositeLit:
			this.out(fmt.Sprintf("(%v*)calloc(1, sizeof(%v));", t2.Type, t2.Type)).outnl()
		default:
			log.Println(reflect.TypeOf(te), t2)
		}
		this.genExpr(scope, te.X)
	case *ast.CompositeLit:
		log.Println(te.Type, te.Elts)
	case *ast.CallExpr:
		this.genExpr(scope, te.Fun)
		this.out("(")
		if len(te.Args) > 0 {
			var tyfmts []string
			for _, e := range te.Args {
				tyfmt := this.exprTypeFmt(scope, e)
				tyfmts = append(tyfmts, "%"+tyfmt)
			}
			this.out(fmt.Sprintf(`"%s"`, strings.Join(tyfmts, " ")))
			this.out(", ")

			for _, e := range te.Args {
				this.genExpr(scope, e)
			}
		}
		this.out(")").outfh().outnl()
	case *ast.BasicLit:
		ety := this.info.TypeOf(e)
		switch t := ety.Underlying().(type) {
		case *types.Basic:
			switch t.Kind() {
			case types.Int:
				this.out(te.Value)
			case types.String:
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
func (this *g2nc) exprType(scope *ast.Scope, e ast.Expr) string {
	switch te := e.(type) {
	case *ast.Ident:
		return te.Name
	case *ast.ArrayType:
	case *ast.StructType:
	case *ast.UnaryExpr:
		return this.exprType(scope, te.X) + "*"
	case *ast.CompositeLit:
		return this.exprType(scope, te.Type)
	default:
		log.Println(reflect.TypeOf(e), te)
	}
	return ""
}
func (this *g2nc) exprTypeFmt(scope *ast.Scope, e ast.Expr) string {
	ety := this.info.TypeOf(e)
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
		fmt.Print(s, " ")
		this.sb.WriteString(s + " ")
	}
	return this
}

func (this *g2nc) code() string {
	return this.sb.String()
}
