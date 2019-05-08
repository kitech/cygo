package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"gopp"
	"log"

	"golang.org/x/tools/go/ast/astutil"
)

type ParserContext struct {
	path     string
	fset     *token.FileSet
	pkgs     map[string]*ast.Package
	typkgs   *types.Package
	conf     types.Config
	info     types.Info
	cursors  map[ast.Node]*astutil.Cursor
	grstargs map[string]bool // goroutines packed arguments structure
}

func NewParserContext(path string) *ParserContext {
	this := &ParserContext{}
	this.path = path
	this.info.Types = make(map[ast.Expr]types.TypeAndValue)
	this.info.Defs = make(map[*ast.Ident]types.Object)
	this.info.Uses = make(map[*ast.Ident]types.Object)
	this.cursors = make(map[ast.Node]*astutil.Cursor)
	this.grstargs = make(map[string]bool)

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
		astutil.Apply(pkg, func(c *astutil.Cursor) bool {
			tc := *c
			this.cursors[c.Node()] = &tc
			return true
		}, func(c *astutil.Cursor) bool {
			return true
		})
	}

	this.conf.DisableUnusedImportCheck = true
	this.conf.Error = func(err error) { log.Println(err) }

	this.typkgs, err = this.conf.Check(this.path, this.fset, files, &this.info)
	log.Println("types:", len(this.info.Types))

	return err
}
