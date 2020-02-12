package main

import (
	"go/ast"
	"go/token"

	"golang.org/x/tools/go/ast/astutil"
)

type TransformContext struct {
	c      *astutil.Cursor // curent cursor
	n      ast.Node        // c.Node
	lastst *astutil.Cursor // last can InsertBefore, or InsertAfter cursor
	ispre  bool
}
type Transformer interface {
	afterchk() bool
	apply(tfctx *TransformContext)
}

var transforms = []Transformer{}

func regtransformer(tfer Transformer) {
	transforms = append(transforms, tfer)
}

// return补全为带名字的
type TfFixRetname struct {
}

type TfTmpvars struct {
}

func init() { regtransformer(&TfTmpvars{}) }

func (tf *TfTmpvars) afterchk() bool { return false }
func (tf *TfTmpvars) apply(ctx *TransformContext) {
	c := ctx.c
	n := ctx.n
	lastst := ctx.lastst

	if ctx.ispre {
		switch te := n.(type) {
		case *ast.CompositeLit:
			// 检查每个元素是否是常量，IDENT
		case *ast.CallExpr:
			// 检查每个元素是否是常量，IDENT
		case *ast.ReturnStmt:
			// 检查每个元素是否是常量，IDENT
			for idx, aex := range te.Results {
				_, isidt := aex.(*ast.Ident)
				if isidt {
					continue
				}
				as := newtmpassign(aex)
				lastst.InsertBefore(as)
				te.Results[idx] = as.Lhs[0]
			}
		case *ast.IfStmt:
			// 检查每个元素是否是常量，IDENT
			if te.Init != nil {
				bs := &ast.BlockStmt{}
				bs.List = append(bs.List, te.Init, te)
				te.Init = nil
				lastst.InsertBefore(bs)
				c.Delete()
			}
		case *ast.SwitchStmt:
			// 检查每个元素是否是常量，IDENT
			if te.Init != nil {
				bs := &ast.BlockStmt{}
				bs.List = append(bs.List, te.Init, te)
				te.Init = nil
				lastst.InsertBefore(bs)
				c.Delete()
			}
		case *ast.ForStmt:
			// 检查每个元素是否是常量，IDENT
			if te.Init != nil {
				idxer := newtmpassign(newLitInt(-1))
				bs := &ast.BlockStmt{}
				bs.List = append(bs.List, idxer, te.Init, te)
				te.Init = nil
				idxpe := &ast.IncDecStmt{Tok: token.INC, X: idxer.Lhs[0]}
				te.Body.List = append([]ast.Stmt{idxpe}, te.Body.List...)
				lastst.InsertBefore(bs)
				c.Delete()
			}
		default:
			_ = te
		}
	} else {

	}
	return
}

func newtmpassign(te ast.Expr) *ast.AssignStmt {
	assign := &ast.AssignStmt{}
	assign.TokPos = te.Pos()
	assign.Tok = token.DEFINE
	idt := newIdent(tmpvarname())
	idt.Obj = ast.NewObj(ast.Var, idt.Name)
	idt.NamePos = te.Pos()
	assign.Lhs = append(assign.Lhs, idt)
	assign.Rhs = append(assign.Rhs, te)
	return assign
}
