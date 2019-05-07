package main

import (
	"fmt"
	"go/ast"
	"log"
	"reflect"
	"strings"
)

type g2nc struct {
	psctx *ParserContext
	sb    strings.Builder
}

func (this *g2nc) genpkgs() {
	// pkgs order?
	for pname, pkg := range this.psctx.pkgs {
		this.genpkg(pname, pkg)
	}
}

func (this *g2nc) genpkg(name string, pkg *ast.Package) {
	log.Println(name)
	for name, f := range pkg.Files {
		this.genfile(name, f)
	}
}
func (this *g2nc) genfile(name string, f *ast.File) {
	log.Println(name)

	// decls order?
	for _, d := range f.Decls {
		this.genDecl(d)
	}
}

func (this *g2nc) genDecl(d ast.Decl) {
	switch td := d.(type) {
	case *ast.FuncDecl:
		this.genFuncDecl(td)
	case *ast.GenDecl:
		this.genGenDecl(td)
	default:
		log.Println("unimplemented", reflect.TypeOf(d))
	}
}

func (this *g2nc) genFuncDecl(d *ast.FuncDecl) {
	log.Println(d.Name)
	this.genFieldList(d.Type.Results, true, false, "")
	this.out(d.Name.String())
	this.out("()").outnl()
	this.genBlockStmt(d.Body)
	this.outnl()
}

func (this *g2nc) genBlockStmt(stmt *ast.BlockStmt) {
	this.out("{").outnl()
	for idx, s := range stmt.List {
		this.genStmt(s, idx)
	}
	this.out("}").outnl()
}

func (this *g2nc) genStmt(stmt ast.Stmt, idx int) {
	switch t := stmt.(type) {
	case *ast.AssignStmt:
		log.Println(t.Tok.String())
		for i := 0; i < len(t.Rhs); i++ {
			this.out(this.exprType(t.Rhs[i]))
			this.genExpr(t.Lhs[i])
			this.out(" = ")
			this.genExpr(t.Rhs[i])
		}
	case *ast.ExprStmt:
		this.genExpr(t.X)
	default:
		log.Println(reflect.TypeOf(stmt), t)
	}
}

func (this *g2nc) genFieldList(flds *ast.FieldList, ovoid bool, withname bool, linebrk string) {
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
		this.genExpr(fld.Type)
		if withname && len(fld.Names) > 0 {
			this.genExpr(fld.Names[0])
		}
		this.out(linebrk)
	}
}

func (this *g2nc) genExpr(e ast.Expr) {
	// log.Println(reflect.TypeOf(e))
	switch te := e.(type) {
	case *ast.Ident:
		log.Println(te.Name, te.String(), te.IsExported(), te.Obj)
		this.out(te.Name, " ")
	case *ast.ArrayType:
		log.Println("unimplemented", te, reflect.TypeOf(e))
	case *ast.StructType:
		this.genFieldList(te.Fields, false, true, ";\n")
	case *ast.UnaryExpr:
		log.Println(te.Op.String(), te.X)
		switch t2 := te.X.(type) {
		case *ast.CompositeLit:
			this.out(fmt.Sprintf("(%v*)calloc(1, sizeof(%v));", t2.Type, t2.Type)).outnl()
		default:
			log.Println(reflect.TypeOf(te), t2)
		}
		this.genExpr(te.X)
	case *ast.CompositeLit:
		log.Println(te.Type, te.Elts)
	case *ast.CallExpr:
		this.genExpr(te.Fun)
		this.out("(")
		if len(te.Args) > 0 {
			var tyfmts []string
			for _, e := range te.Args {
				tyfmt := this.exprTypeFmt(e)
				tyfmts = append(tyfmts, "%"+tyfmt)
			}
			this.out(fmt.Sprintf(`"%s"`, strings.Join(tyfmts, " ")))
			this.out(", ")

			for _, e := range te.Args {
				this.genExpr(e)
			}
		}
		this.out(")").outfh().outnl()
	default:
		log.Println(reflect.TypeOf(e), te)
	}
}
func (this *g2nc) exprType(e ast.Expr) string {
	switch te := e.(type) {
	case *ast.Ident:
		return te.Name
	case *ast.ArrayType:
	case *ast.StructType:
	case *ast.UnaryExpr:
		return this.exprType(te.X) + "*"
	case *ast.CompositeLit:
		return this.exprType(te.Type)
	default:
		log.Println(reflect.TypeOf(e), te)
	}
	return ""
}
func (this *g2nc) exprTypeFmt(e ast.Expr) string {
	switch te := e.(type) {
	case *ast.Ident:
		log.Println(te.Obj.Type, te.String())
		return "s"
	case *ast.ArrayType:
	case *ast.StructType:
	case *ast.UnaryExpr:
		return "p"
	case *ast.CompositeLit:
		return this.exprType(te.Type)
	default:
		log.Println(reflect.TypeOf(e), te)
	}
	return ""
}

func (this *g2nc) genGenDecl(d *ast.GenDecl) {
	log.Println(d.Tok, d.Specs, len(d.Specs), d.Tok.IsKeyword(), d.Tok.IsLiteral(), d.Tok.IsOperator())
	log.Println(reflect.TypeOf(d.Specs))
	for _, spec := range d.Specs {
		log.Println(reflect.TypeOf(spec))
		switch tspec := spec.(type) {
		case *ast.TypeSpec:
			this.genTypeSpec(tspec)
		default:
			log.Println(reflect.TypeOf(d))
		}
	}
}
func (this *g2nc) genTypeSpec(spec *ast.TypeSpec) {
	log.Println(spec.Name, spec.Type)
	this.out("typedef struct {")
	this.outnl()
	this.genExpr(spec.Type)
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
