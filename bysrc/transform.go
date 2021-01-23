package main

import (
	"bytes"
	"go/ast"
	"go/printer"
	"go/token"
	"gopp"
	"log"
	"strings"

	"github.com/thoas/go-funk"
	"golang.org/x/tools/go/ast/astutil"
)

var findupinsable func(cursors map[ast.Node]*astutil.Cursor,
	e ast.Node, lvl int) *astutil.Cursor

func isinsablenode(pe ast.Node) bool {
	if _, ok := pe.(*ast.BlockStmt); ok {
		return true
	}
	if _, ok := pe.(*ast.CaseClause); ok {
		return true
	}
	if _, ok := pe.(*ast.File); ok {
		return true
	}
	return false
}
func init() {
	findupinsable = func(cursors map[ast.Node]*astutil.Cursor,
		e ast.Node, lvl int) *astutil.Cursor {
		c := cursors[e]
		if c == nil {
			log.Println(lvl, reftyof(e))
			return nil
		}
		pe := c.Parent()
		if pe == nil {
			log.Println(lvl, reftyof(e), reftyof(pe))
			return nil
		}
		if _, ok := pe.(*ast.BlockStmt); ok {
			return c
		}
		if _, ok := pe.(*ast.CaseClause); ok {
			return c
		}
		if _, ok := pe.(*ast.File); ok {
			return c
		}
		return findupinsable(cursors, pe, lvl+1)
	}
}

type TransformContext struct {
	c     *astutil.Cursor // curent cursor
	n     ast.Node        // c.Node
	s     *astutil.Cursor // current stmt
	ispre bool
	pc    *ParserContext
	fio   *ast.File // current file
	cycle int

	inslines map[ast.Node][]ast.Stmt
}

func newTransformContext(pc *ParserContext, c *astutil.Cursor, ispre bool) *TransformContext {
	ctx := &TransformContext{}
	ctx.reset(pc, c, ispre)
	return ctx
}
func (ctx *TransformContext) reset(pc *ParserContext, c *astutil.Cursor, ispre bool) {
	ctx.inslines = map[ast.Node][]ast.Stmt{}
	ctx.c = c
	ctx.n = c.Node()
	ctx.ispre = ispre
	ctx.pc = pc
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
				if _, ok := te.Lhs[0].(*ast.IndexExpr); ok {
					if _, ok2 := te.Rhs[0].(*ast.Ident); !ok2 {
						// right non ident expr to ident
						as := newtmpassign(te.Rhs[0])
						te.Rhs[0] = as.Lhs[0]
						ctx.addline(te, as)
					}
				}
			} else {
				//log.Panicln("noimpl", len(te.Lhs), len(te.Rhs))
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
		case *ast.SliceExpr:
			if _, ok := te.Low.(*ast.Ident); !ok && te.Low != nil {
				as := newtmpassign(te.Low)
				te.Low = as.Lhs[0]
				ctx.addline(te, as)
			}
			if _, ok := te.High.(*ast.Ident); !ok && te.High != nil {
				as := newtmpassign(te.High)
				te.High = as.Lhs[0]
				ctx.addline(te, as)
			}
			if _, ok := te.Max.(*ast.Ident); !ok && te.Max != nil {
				as := newtmpassign(te.Max)
				te.Max = as.Lhs[0]
				ctx.addline(te, as)
			}
			if _, ok := te.X.(*ast.Ident); !ok {
				as := newtmpassign(te.X)
				te.X = as.Lhs[0]
				ctx.addline(te, as)
			}
		case *ast.IndexExpr:
			// index key to ident
			// moved to seperate cycle
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

/// call args to ident
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
				if idt.Name == "ifelse" {
					// log.Println("got ifelse", reftyof(c.Parent()))
					var lexpr ast.Expr
					pn := c.Parent()
					switch pnty := pn.(type) {
					case *ast.ExprStmt: // dont care value
					case *ast.AssignStmt: // need value
						gopp.Assert(len(pnty.Lhs) == 1, "not support")
						lexpr = pnty.Lhs[0]
					default:
						log.Panicln("not support", reftyof(pnty))
					}
					// 看是否能够查找出来表达式的值类型
					var valty ast.Expr // 用于在 if () {} else {} 之外声明变量
					for i := 1; i < len(te.Args); i++ {
						ae := te.Args[i]
						valty = exprtype(ae)
						if valty != nil {
							break
						}
					}
					// 简单语法检查
					if lexpr != nil {
						gopp.Assert(valty != nil, "wtfff", reftyof(te.Args[1]))
					}

					// if 语句之前的临时赋值变量
					tmpidt := newIdent(tmpvarname())
					tmpval := &ast.ValueSpec{}
					tmpval.Type = valty
					tmpval.Names = append(tmpval.Names, tmpidt)

					// 转换成if语句
					ifst := &ast.IfStmt{}
					ifst.Cond = te.Args[0]
					ifbody := &ast.BlockStmt{}
					ifst.Body = ifbody
					elbody := &ast.BlockStmt{}
					ifst.Else = elbody

					// 填充 if/else分支语句
					if lexpr != nil {
						as1 := newtmpassign(te.Args[1])
						as2 := newtmpassign(te.Args[2])
						ifbody.List = append(ifbody.List, as1)
						elbody.List = append(elbody.List, as2)
						as3 := newassign2(tmpidt, as1.Lhs[0])
						as3.Tok = token.ASSIGN
						as4 := newassign2(tmpidt, as2.Lhs[0])
						as4.Tok = token.ASSIGN
						ifbody.List = append(ifbody.List, as3)
						elbody.List = append(elbody.List, as4)
					} else {
						exprst1 := &ast.ExprStmt{}
						exprst2 := &ast.ExprStmt{}
						exprst1.X = te.Args[1]
						exprst2.X = te.Args[2]
						ifbody.List = append(ifbody.List, exprst1)
						elbody.List = append(elbody.List, exprst2)
					}

					if false {
						log.Println(astnodestr(ctx.pc.fset, ifst, true))
					}

					tmpgend := &ast.GenDecl{}
					tmpgend.Tok = token.VAR
					tmpgend.Specs = append(tmpgend.Specs, tmpval)
					tmpdecl := &ast.DeclStmt{}
					tmpdecl.Decl = tmpgend
					if lexpr != nil {
						ctx.addline(c.Parent(), tmpdecl)
					}
					ctx.addline(c.Parent(), ifst)
					// 替换当前表达式
					if lexpr != nil {
						c.Replace(tmpidt)
					} else {
						nop := &ast.BinaryExpr{Op: token.EQL}
						nop.X = trueidt
						nop.Y = trueidt
						c.Replace(nop)
					}

					break
				}
				if funk.Contains([]string{"make"}, idt.Name) {
					skip0 = true
				}
			} else if fnlit, ok := te.Fun.(*ast.FuncLit); ok {
				as := newtmpassign(fnlit)
				te.Fun = as.Lhs[0]
				ctx.addline(te, as)
			} else if selo, ok := te.Fun.(*ast.SelectorExpr); ok {
				xidt, ok1 := selo.X.(*ast.Ident)
				if ok1 && xidt.Obj != nil && xidt.Obj.Kind == ast.Typ {
					// tyfoo.bar() 静态调用 => (tyfoo{}).bar()
					compexpr := &ast.CompositeLit{Type: xidt}
					andexpr := &ast.UnaryExpr{Op: token.AND, X: compexpr}
					starexpr := &ast.StarExpr{X: &ast.ParenExpr{X: andexpr}}
					starexpr.Star = te.Pos()
					as := newtmpassign(starexpr)
					selo.X = as.Lhs[0]
					ctx.addline(te, as)
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

/// add new0 method for struct to ast
type TfTmpvars3 struct {
}

func init() { regtransformer(&TfTmpvars3{}) }

func (tf *TfTmpvars3) afterchk() bool { return false }
func (tf *TfTmpvars3) apply(ctx *TransformContext) {
	c := ctx.c
	n := ctx.n
	_ = c

	if ctx.cycle > 0 {
		return
	}
	if ctx.ispre {
		switch te := n.(type) {
		case *ast.TypeSpec:
			if _, ok := te.Type.(*ast.StructType); ok {
				log.Println(te.Name, reftyof(te.Type), ctx.fio.Name)
				// insert after
				newfn := &ast.FuncDecl{}
				newfn.Name = newIdent("new0")
				rcv := &ast.Field{}
				rcv.Type = te.Name

				rete := &ast.StarExpr{}
				rete.X = te.Name
				newfn.Recv = &ast.FieldList{}
				newfn.Recv.List = append(newfn.Recv.List, rcv)
				reto := &ast.Field{}
				reto.Type = rete

				sig := &ast.FuncType{}
				sig.Func = te.Pos()
				sig.Params = &ast.FieldList{}
				sig.Results = &ast.FieldList{}
				sig.Results.List = append(sig.Results.List, reto)
				newfn.Type = sig

				body := &ast.BlockStmt{}
				body.Lbrace = te.Pos()
				newfn.Body = body
				retexpr := &ast.ReturnStmt{}
				as := newtmpassign(&ast.UnaryExpr{
					Op: token.AND, X: &ast.CompositeLit{Type: te.Name},
				})
				as.TokPos = te.Pos()
				retexpr.Results = append(retexpr.Results, as.Lhs[0])
				retexpr.Return = te.Pos()
				body.List = append(body.List, as, retexpr)

				ds := &ast.DeclStmt{}
				ds.Decl = newfn

				if false {
					buf := bytes.NewBuffer(nil)
					err := printer.Fprint(buf, ctx.pc.fset, newfn)
					bufcc := string(buf.Bytes())
					bufcc = strings.ReplaceAll(bufcc, "\n", " ")
					log.Println(err, len(bufcc), bufcc)
				}
				if false {
					ctx.addline(te, ds)
				}
				if true {
					fio := ctx.fio
					fio.Decls = append(fio.Decls, ds.Decl)
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

/// index key to ident
type TfTmpvars4 struct {
}

func init() { regtransformer(&TfTmpvars4{}) }

func (tf *TfTmpvars4) afterchk() bool { return false }
func (tf *TfTmpvars4) apply(ctx *TransformContext) {
	c := ctx.c
	n := ctx.n
	_ = c

	if ctx.ispre {
		switch te := n.(type) {
		case *ast.IndexExpr:
			// index key to ident
			if _, ok := te.Index.(*ast.Ident); !ok {
				if true {
					as := newtmpassign(te.Index)
					te.Index = as.Lhs[0]
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

///
func InsertBefore(pc *ParserContext, c, inscs *astutil.Cursor, stmts []ast.Stmt) {
	if false {
		log.Println(inscs != nil, inscs.Index(), reftyof(inscs.Node()))
	}
	if gend, ok := inscs.Node().(*ast.GenDecl); ok {
		log.Println("got some", gend.Tok, len(stmts), exprpos(pc, gend))
		valsp := &ast.ValueSpec{}
		_ = valsp
		var newspecs []ast.Spec
		oldcnt := len(gend.Specs)
		for idx, spx := range gend.Specs {
			log.Println(idx, reftyof(spx), reftyof(c.Parent()), reftyof(c.Node()))
			if spx == c.Parent() {
				log.Println("got it")
				for _, stmt := range stmts {
					as := stmt.(*ast.AssignStmt)
					valsp := &ast.ValueSpec{}
					valsp.Names = append(valsp.Names, as.Lhs[0].(*ast.Ident))
					valsp.Values = append(valsp.Values, as.Rhs[0])
					newspecs = append(newspecs, valsp)
				}
			} else if spx == c.Node() {
				log.Println("wtttt")
			}

			newspecs = append(newspecs, spx)
		}
		if len(newspecs) > oldcnt {
			gend.Specs = newspecs
		}
		return
	}
	curcs := inscs.Node().(ast.Stmt)
	var oldlst []ast.Stmt
	switch vec := inscs.Parent().(type) {
	case *ast.BlockStmt:
		oldlst = vec.List
	case *ast.CaseClause:
		oldlst = vec.Body
	default:
		log.Panicln(reftyof(vec))
	}
	oldcnt := len(oldlst)

	var lst []ast.Stmt
	for _, stmt := range oldlst {
		if stmt == curcs {
			lst = append(lst, stmts...)
		}
		lst = append(lst, stmt)
	}
	if len(lst) > oldcnt {
		switch vec := inscs.Parent().(type) {
		case *ast.BlockStmt:
			vec.List = lst
		case *ast.CaseClause:
			vec.Body = lst
		default:
			log.Panicln(reftyof(vec))
		}
		// blkst.List = lst
		// log.Println("change", len(stmts), len(lst))
	} else {
		log.Println("not change", len(stmts))
	}
}

///

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
	// log.Printf("var %#v = %#v\n", idt, te)
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
