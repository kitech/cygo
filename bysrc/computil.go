package main

import (
	"go/ast"
	"go/types"
	"strings"
)

// compile line context
type compContext struct {
	le  ast.Expr
	re  ast.Expr
	lty types.Type
	rty types.Type
}

func ismapty(tystr string) bool {
	return strings.HasPrefix(tystr, "map[")
}
