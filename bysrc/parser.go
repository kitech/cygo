package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"gopp"
)

type ParserContext struct {
	path string
	fset *token.FileSet
	pkgs map[string]*ast.Package
}

func NewParserContext(path string) *ParserContext {
	this := &ParserContext{}
	this.path = path

	return this
}

func (this *ParserContext) Init() error {
	this.fset = token.NewFileSet()
	pkgs, err := parser.ParseDir(this.fset, this.path, nil, 0|parser.AllErrors)
	gopp.ErrPrint(err)
	this.pkgs = pkgs
	return err
}
