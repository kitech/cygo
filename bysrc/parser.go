package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"gopp"
	"log"
)

type ParserContext struct {
	path   string
	fset   *token.FileSet
	pkgs   map[string]*ast.Package
	typkgs *types.Package
	conf   types.Config
	info   types.Info
}

func NewParserContext(path string) *ParserContext {
	this := &ParserContext{}
	this.path = path
	this.info.Types = make(map[ast.Expr]types.TypeAndValue)
	this.info.Defs = make(map[*ast.Ident]types.Object)
	this.info.Uses = make(map[*ast.Ident]types.Object)

	return this
}

func (this *ParserContext) Init() error {
	this.fset = token.NewFileSet()
	pkgs, err := parser.ParseDir(this.fset, this.path, nil, 0|parser.AllErrors)
	gopp.ErrPrint(err)
	this.pkgs = pkgs

	var files []*ast.File
	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			files = append(files, file)
		}
	}

	this.typkgs, err = this.conf.Check(this.path, this.fset, files, &this.info)
	log.Println("types:", len(this.info.Types))

	return err
}
