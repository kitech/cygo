package main

import (
	"fmt"
	"go/ast"
	"go/token"
	"log"
	"path/filepath"
	"reflect"
	"strconv"

	"golang.org/x/tools/go/ast/astutil"
)

// 带 scope的遍历
func Visit(node ast.Node, pre func(c *astutil.Cursor) bool, post func(c *astutil.Cursor) bool) (result ast.Node) {
	// ast.Visitor.Visit(node ast.Node)

	return nil
}

func find_use_ident(pc *ParserContext, node ast.Node, idt *ast.Ident) []ast.Node {
	res := []ast.Node{}
	astutil.Apply(node, func(c *astutil.Cursor) bool {
		switch te := c.Node().(type) {
		case *ast.SelectorExpr:
			teidt, ok := te.X.(*ast.Ident)
			if !ok {
				break
			}
			if teidt.Name == idt.Name {
				log.Println(teidt, idt, teidt == idt, te.Sel)
				res = append(res, te)
			}
		}
		return true
	}, nil)
	return res
}

// return true for found, false goon
func upfind_func(pc *ParserContext, cs *astutil.Cursor, no int,
	f func(c *astutil.Cursor) bool) ast.Node {
	if cs == nil {
		return nil
	}
	pn := cs.Parent()
	if pn == nil {
		return nil
	}
	pcs := pc.cursors[pn]
	if pcs == nil {
		return nil
	}
	if ok := f(pcs); ok {
		return pn
	}
	return upfind_func(pc, pcs, no+1, f)
}

func upfind_blockstmt(pc *ParserContext, cs *astutil.Cursor, no int) *ast.BlockStmt {
	n := upfind_func(pc, cs, no, func(c *astutil.Cursor) bool {
		_, ok := c.Node().(*ast.BlockStmt)
		return ok
	})
	if n == nil {
		return nil
	}
	return n.(*ast.BlockStmt)
}

func (pc *ParserContext) dumpup(cs *astutil.Cursor, no int) {
	if cs == nil {
		return
	}
	log.Println(no, cs.Name(), reflect.TypeOf(cs.Node()))
	pn := cs.Parent()
	pcs := pc.cursors[pn]
	pc.dumpup(pcs, no+1)
}

func upfindstmt(pc *ParserContext, cs *astutil.Cursor, no int) ast.Stmt {
	if cs == nil {
		return nil
	}

	pn := cs.Parent()
	pcs := pc.cursors[pn]
	if stmt, ok := pn.(ast.Stmt); ok {
		return stmt
	} else {
		return upfindstmt(pc, pcs, no+1)
	}
}

func upfindFuncDeclNode(pc *ParserContext, n ast.Node, no int) *ast.FuncDecl {
	cs := pc.cursors[n]
	return upfindFuncDecl(pc, cs, no)
}
func upfindFuncDeclAst(pc *ParserContext, e ast.Expr, no int) *ast.FuncDecl {
	cs := pc.cursors[e]
	return upfindFuncDecl(pc, cs, no)
}
func upfindFuncDecl(pc *ParserContext, cs *astutil.Cursor, no int) *ast.FuncDecl {
	if cs == nil {
		return nil
	}
	pn := cs.Parent()
	pcs := pc.cursors[pn]
	if stmt, ok := pn.(*ast.FuncDecl); ok {
		return stmt
	} else {
		return upfindFuncDecl(pc, pcs, no+1)
	}
}

func newimpspec(path string, name string) *ast.ImportSpec {
	// &ast.ImportSpec{Path: &ast.BasicLit{Value: strconv.Quote("path/to/pkg")}}
	if name == "" {
		name = filepath.Base(path)
	}
	impspec := &ast.ImportSpec{}
	impspec.Name = ast.NewIdent(name)
	impspec.Path = &ast.BasicLit{}
	impspec.Path.Kind = token.STRING
	impspec.Path.Value = strconv.Quote(path)
	impspec.Path.ValuePos = token.NoPos
	return impspec
}

func newIdent(v string) *ast.Ident {
	idt := &ast.Ident{}
	idt.Name = v
	idt.NamePos = token.NoPos
	return idt
}

func newIdentp(v string, p token.Pos) *ast.Ident {
	idt := &ast.Ident{}
	idt.Name = v
	idt.NamePos = token.NoPos
	idt.NamePos = p
	return idt
}

func newVardecl(name string, typ ast.Expr, pos token.Pos) (*ast.Ident, *ast.DeclStmt) {
	idt := newIdent(name)
	idt.NamePos = pos
	idt.Obj = ast.NewObj(ast.Var, idt.Name)

	valspec := &ast.ValueSpec{}
	valspec.Type = typ
	valspec.Names = append(valspec.Names, idt)
	gendecl := &ast.GenDecl{}
	gendecl.Tok = token.VAR
	gendecl.TokPos = pos
	gendecl.Specs = append(gendecl.Specs, valspec)
	declvar := &ast.DeclStmt{}
	declvar.Decl = gendecl

	return idt, declvar
}

func newLitInt(v int) *ast.BasicLit {
	return &ast.BasicLit{Kind: token.INT, Value: fmt.Sprintf("%d", v)}
}
func newLitStr(v string) *ast.BasicLit {
	return &ast.BasicLit{Kind: token.STRING, Value: fmt.Sprintf("%s", v)}
}
func newLitFloat(v float32) *ast.BasicLit {
	return &ast.BasicLit{Kind: token.FLOAT, Value: fmt.Sprintf("%f", v)}
}

var trueidt = newIdent("true")
var falseidt = newIdent("false")
var nilidtcon = newIdent("nil")

func newLitBool(v bool) *ast.Ident {
	if v {
		return trueidt
	}
	return falseidt
}
