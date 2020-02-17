package main

import (
	"go/ast"
	"go/token"
	"log"
)

// 尝试计算 表达式的类型，ast.Expr 表示形式
func exprtype(ae ast.Expr) ast.Expr {
	var valty ast.Expr
	switch aty := ae.(type) {
	case *ast.BasicLit:
		switch aty.Kind {
		case token.STRING:
			valty = newIdent("string")
		case token.INT:
			valty = newIdent("int")
		case token.FLOAT:
			valty = newIdent("float64")
		default:
			log.Panicln("notimpl", aty.Kind)
		}
	case *ast.Ident:
		obj := aty.Obj
		if obj == nil {
			break
		}
		if obj.Type != nil {
			valty = obj.Type.(ast.Expr)
			break
		}
		if valty == nil && obj.Decl != nil {
			switch declty := obj.Decl.(type) {
			case *ast.Field:
				valty = declty.Type
			case *ast.AssignStmt:
				valty = exprtype(declty.Rhs[0])
			}
		}
		log.Println(aty.Name, obj, obj.Decl, reftyof(obj.Decl))
	case *ast.BinaryExpr:
		tmpty1 := exprtype(aty.X)
		tmpty2 := exprtype(aty.Y)
		/// gopp.Assert(tmpty1)
		if tmpty1 != nil {
			valty = tmpty1
		} else {
			valty = tmpty2
		}
	case *ast.SelectorExpr:
		valty = exprtype(aty.Sel)
	default:
		log.Panicln("notimpl", reftyof(aty))
	}
	return valty
}
