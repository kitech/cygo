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
	}
}

func (this *g2nc) genFuncDecl(d *ast.FuncDecl) {
	log.Println(d.Name)
	this.genFieldList(d.Type.Results, true)
	this.out(d.Name.String())
	this.out("()")
	this.outnl()
	this.out("{}")
	this.outnl()
}

func (this *g2nc) genFieldList(flds *ast.FieldList, ovoid bool) {
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
	default:
		log.Println(reflect.TypeOf(e))
	}
}

func (this *g2nc) outeq() { this.out("=") }
func (this *g2nc) outnl() { this.out("\n") }
func (this *g2nc) out(ss ...string) {
	for _, s := range ss {
		fmt.Print(s, " ")
		this.sb.WriteString(s + " ")
	}
}

func (this *g2nc) code() string {
	return this.sb.String()
}
