package main

import (
	"go/ast"
	"go/token"
	"log"

	"github.com/thoas/go-funk"
	"golang.org/x/tools/go/ast/astutil"
)

type TransformContext struct {
	c     *astutil.Cursor // curent cursor
	n     ast.Node        // c.Node
	s     *astutil.Cursor // current stmt
	ispre bool
	pc    *ParserContext

	inslines map[ast.Node][]ast.Stmt
}

func newTransformContext(pc *ParserContext, c *astutil.Cursor, ispre bool) *TransformContext {
	ctx := &TransformContext{}
	ctx.inslines = map[ast.Node][]ast.Stmt{}
	ctx.c = c
	ctx.n = c.Node()
	ctx.ispre = ispre
	ctx.pc = pc
	return ctx
}
func (ctx *TransformContext) addline(ce ast.Node, line ast.Stmt) {
	ctx.inslines[ce] = append(ctx.inslines[ce], line)
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

///
type TfTmpvars struct {
}

func init() { regtransformer(&TfTmpvars{}) }

func (tf *TfTmpvars) afterchk() bool { return false }
func (tf *TfTmpvars) apply(ctx *TransformContext) {
	c := ctx.c
	n := ctx.n
	_ = c

	if ctx.ispre {
		switch te := n.(type) {
		case *ast.Ident:
			if te.Name == "_" {
				// c.Replace(newIdent(tmpvarname()))
			}
		case *ast.AssignStmt:
			// _ = x => tmp := x
			underline_cnt := 0
			for idx, ae := range te.Lhs {
				if idt, ok := ae.(*ast.Ident); ok && idt.Name == "_" {
					te.Lhs[idx] = newIdentp(tmpvarname(), ae.Pos())
					underline_cnt++
				}
			}
			if underline_cnt == len(te.Lhs) {
				te.Tok = token.DEFINE
			}

			// // x,y := 1, 2 => x := 1; y := 2
			if len(te.Lhs) > 1 && len(te.Rhs) == len(te.Lhs) {
				for idx, _ := range te.Lhs {
					as := newtmpassign(te.Rhs[idx])
					te.Rhs[idx] = as.Lhs[0]
					// lastst.InsertBefore(as)
					ctx.addline(te, as)
				}
				cnt := len(te.Lhs)
				for idx := 0; idx < cnt-1; idx++ {
					ae := te.Lhs[idx]
					as := newassign2(ae, te.Rhs[idx])
					as.Tok = te.Tok
					ctx.addline(te, as)
				}
				te.Lhs = te.Lhs[cnt-1:]
				te.Rhs = te.Rhs[cnt-1:]
			} else if len(te.Lhs) == 1 && len(te.Rhs) == len(te.Lhs) {
				if _, ok := te.Rhs[0].(*ast.Ident); !ok {
				}
			}
		case *ast.CompositeLit:
			// 检查每个元素是否是常量，IDENT
			for idx, eex := range te.Elts {
				switch ee := eex.(type) {
				case *ast.KeyValueExpr:
					if _, ok := ee.Value.(*ast.Ident); !ok {
						as := newtmpassign(ee.Value)
						ee.Value = as.Lhs[0]
						ctx.addline(te, as)
					}
				default:
					if _, ok := eex.(*ast.Ident); !ok {
						as := newtmpassign(eex)
						te.Elts[idx] = as.Lhs[0]
						ctx.addline(te, as)
					}
				}
			}

			// TODO astruct{1,2,3} = > astruct{}; a.x = 1, a.y = 2; a.z = 3

		case *ast.CallExpr:
		case *ast.ReturnStmt:
			// 检查每个元素是否是常量，IDENT
			for idx, aex := range te.Results {
				_, isidt := aex.(*ast.Ident)
				if isidt {
					continue
				}
				as := newtmpassign(aex)
				te.Results[idx] = as.Lhs[0]
				ctx.addline(te, as)
			}
		case *ast.IfStmt:
			// 检查每个元素是否是常量，IDENT
			initst := te.Init
			condst := te.Cond

			// if a:=1; a {} => { a:=1; if a {} }
			_, simcond := condst.(*ast.Ident)
			if te.Init != nil {
				te.Init = nil
				bs := &ast.BlockStmt{}
				bs.List = append(bs.List, initst, te)
				c.Replace(bs)
			} else if !simcond {
				as := newtmpassign(condst)
				te.Cond = as.Lhs[0]
				ctx.addline(te, as)
			}

		case *ast.SwitchStmt:
			// 检查每个元素是否是常量，IDENT
			// switch a:=11; a {} => { a:=1; switch a{} }
			initst := te.Init

			if initst != nil {
				te.Init = nil
				bs := &ast.BlockStmt{}
				bs.List = append(bs.List, initst, te)
				c.Replace(bs)
			}
		case *ast.ForStmt:
			// 检查每个元素是否是常量，IDENT
			initst := te.Init
			condst := te.Cond

			// for a:=1;xxx;yyy {} => { a:=1; for ;xxx;yyy;{} break }
			_, _ = initst, condst
			if initst != nil {
				te.Init = nil
				bs := &ast.BlockStmt{}
				bs.List = append(bs.List, initst, te)
				c.Replace(bs)
			}
		default:
			_ = te
		}
	} else {
		switch te := n.(type) {
		case *ast.CallExpr:
			// 检查每个元素是否是常量，IDENT
		default:
			_ = te
		}
	}
	return
}

///
type TfTmpvars2 struct {
}

func init() { regtransformer(&TfTmpvars2{}) }

func (tf *TfTmpvars2) afterchk() bool { return false }
func (tf *TfTmpvars2) apply(ctx *TransformContext) {
	c := ctx.c
	n := ctx.n
	_ = c

	if ctx.ispre {
		switch te := n.(type) {
		case *ast.CallExpr:
			// 检查每个元素是否是常量，IDENT
			skip0 := false
			if idt, ok := te.Fun.(*ast.Ident); ok {
				if funk.Contains([]string{"make"}, idt.Name) {
					skip0 = true
				}
			}
			for idx, ae := range te.Args {
				if idx == 0 && skip0 {
					continue
				}
				if _, ok := ae.(*ast.Ident); !ok {
					as := newtmpassign(ae)
					te.Args[idx] = as.Lhs[0]
					ctx.addline(te, as)
				}
			}
		default:
			_ = te
		}
	} else {
		switch te := n.(type) {
		default:
			_ = te
		}
	}
	return
}

func InsertBefore(c *astutil.Cursor, n ast.Node) {
	cn := c.Node()
	pn := c.Parent()
	log.Println(reftyof(cn), reftyof(pn), c.Index())
	if blks, ok := pn.(*ast.BlockStmt); ok {
		var lst []ast.Stmt
		for idx, stmt := range blks.List {
			log.Println(idx, stmt == cn)
			if stmt == cn {
				lst = append(lst, n.(ast.Stmt))
			}
			lst = append(lst, stmt)
		}
		if len(lst) > len(blks.List) {
			blks.List = lst
		}
	}
}

// 当前node在 parent中的索引号
func GetCursorIndex(c *astutil.Cursor) int {
	pidx := -1
	cn := c.Node()
	pn := c.Parent()
	if blks, ok := pn.(*ast.BlockStmt); ok {
		for idx, stmt := range blks.List {
			if stmt == cn {
				return idx
			}
		}
	}
	return pidx
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

func newassign2(le, te ast.Expr) *ast.AssignStmt {
	assign := &ast.AssignStmt{}
	assign.TokPos = te.Pos()
	assign.Tok = token.DEFINE
	assign.Lhs = append(assign.Lhs, le)
	assign.Rhs = append(assign.Rhs, te)
	return assign
}
