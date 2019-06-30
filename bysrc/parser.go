package main

import (
	"go/ast"
	"go/build"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"gopp"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"golang.org/x/tools/go/ast/astutil"

	"github.com/twmb/algoimpl/go/graph"
)

type ParserContext struct {
	path     string
	fset     *token.FileSet
	pkgs     map[string]*ast.Package
	files    []*ast.File
	typkgs   *types.Package
	conf     types.Config
	info     types.Info
	cursors  map[ast.Node]*astutil.Cursor
	grstargs map[string]bool // goroutines packed arguments structure

	typeDeclsm    map[string]*ast.TypeSpec
	typeDeclsv    []*ast.TypeSpec
	funcDeclsm    map[string]*ast.FuncDecl
	funcDeclsv    []*ast.FuncDecl
	funcdeclNodes map[string]graph.Node

	gb     *graph.Graph
	bdpkgs *build.Package
	ccode  string
}

func NewParserContext(path string) *ParserContext {
	this := &ParserContext{}
	this.path = path
	this.info.Types = make(map[ast.Expr]types.TypeAndValue)
	this.info.Defs = make(map[*ast.Ident]types.Object)
	this.info.Uses = make(map[*ast.Ident]types.Object)
	this.cursors = make(map[ast.Node]*astutil.Cursor)
	this.grstargs = make(map[string]bool)
	this.typeDeclsm = make(map[string]*ast.TypeSpec)
	this.funcDeclsm = make(map[string]*ast.FuncDecl)
	this.funcdeclNodes = make(map[string]graph.Node)
	this.gb = graph.New(graph.Directed)

	return this
}

func (this *ParserContext) Init() error {
	bdpkgs, err := build.ImportDir(this.path, build.ImportComment)
	gopp.ErrPrint(err)
	this.bdpkgs = bdpkgs
	if len(bdpkgs.InvalidGoFiles) > 0 {
		log.Fatalln("Have InvalidGoFiles", bdpkgs.InvalidGoFiles)
	}
	// TODO use go-clang to resolve c function signature
	// TODO extract c code from bdpkgs.CgoFiles

	this.fset = token.NewFileSet()
	pkgs, err := parser.ParseDir(this.fset, this.path, this.dirFilter, 0|parser.AllErrors|parser.ParseComments)
	gopp.ErrPrint(err)
	this.pkgs = pkgs

	this.ccode = this.pickCCode()
	this.walkpass0()
	files := this.files

	this.conf.DisableUnusedImportCheck = true
	this.conf.Error = func(err error) {
		if !strings.Contains(err.Error(), "declared but not used") {
			log.Println(err)
		}
	}
	this.conf.FakeImportC = true
	this.conf.Importer = &mypkgimporter{}

	this.typkgs, err = this.conf.Check(this.path, this.fset, files, &this.info)

	this.walkpass1()
	log.Println("pkgs", this.typkgs.Name(), "types:", len(this.info.Types),
		"typedefs", len(this.typeDeclsm), "funcdefs", len(this.funcDeclsm))

	nodes := this.gb.TopologicalSort()
	for _, node := range nodes {
		this.funcDeclsv = append(this.funcDeclsv, this.funcDeclsm[(*node.Value).(string)])
	}
	// unused decls
	for _, d := range this.funcDeclsm {
		if _, ok := builtinfns[d.Name.Name]; ok {
			continue
		}
		invec := false
		for _, d1 := range this.funcDeclsv {
			if d1 == d {
				invec = true
				break
			}
		}
		if !invec {
			this.funcDeclsv = append(this.funcDeclsv, d)
		}
	}

	return err
}

func (this *ParserContext) nameFilter2(filename string, files []string) bool {
	for _, okfile := range files {
		if filename == okfile {
			return true // keep
		}
	}
	return false
}
func (this *ParserContext) nameFilter(filename string) bool {
	if this.nameFilter2(filename, this.bdpkgs.GoFiles) {
		return true
	}
	if this.nameFilter2(filename, this.bdpkgs.CgoFiles) {
		return true
	}
	return false
}
func (this *ParserContext) dirFilter(f os.FileInfo) bool {
	return this.nameFilter(f.Name())
}

type mypkgimporter struct{}

func (this *mypkgimporter) Import(path string) (pkgo *types.Package, err error) {
	if true {
		// go 1.12
		fset := token.NewFileSet()
		pkgo, err = importer.ForCompiler(fset, "source", nil).Import(path)
	} else {
		pkgo, err = importer.Default().Import(path)
	}
	gopp.ErrPrint(err, path)
	return pkgo, err
}

func (p *ParserContext) exprpos(e ast.Node) token.Position {
	return p.fset.Position(e.Pos())
}

func (this *ParserContext) pickCCode() string {
	rawcode := this.pickCCode2()
	lines := strings.Split(rawcode, "\n")
	rawcode = ""
	for _, line := range lines {
		if !strings.HasPrefix(line, "#cgo ") {
			rawcode += line + "\n"
		}
	}
	// log.Println("got c code", rawcode)
	return rawcode
}
func (this *ParserContext) pickCCode2() string {
	ccode := ""
	for _, f := range this.bdpkgs.CgoFiles {
		var fo *ast.File = this.findFileobj(f)
		ccode += this.pickCCode3(fo)
	}
	return ccode
}
func (this *ParserContext) pickCCode3(fo *ast.File) string {
	for idx, cmto := range fo.Comments {
		// isimpcblock(cmto)???
		for idx2, impo := range fo.Imports {
			gopp.G_USED(idx, idx2)
			if impo.Pos()-token.Pos(len("\nimport ")) == cmto.End() {
				// log.Println("got c code", cmto.Text())
				return cmto.Text()
			}
		}
	}
	return ""
}
func (this *ParserContext) findFileobj(fbname string) *ast.File {
	for _, pkgo := range this.pkgs {
		for filename, fileo := range pkgo.Files {
			name := filepath.Base(filename)
			if name == fbname {
				return fileo
			}
		}
	}
	return nil
}

func (pc *ParserContext) walkpass0() {
	this := pc
	pkgs := pc.pkgs

	var files []*ast.File
	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			if strings.HasSuffix(file.Name.Name, "_test") {
				continue
			}
			files = append(files, file)
		}
	}
	this.files = files
}

func (pc *ParserContext) walkpass1() {
	this := pc
	pkgs := pc.pkgs

	for _, pkg := range pkgs {

		var curfds []string // stack, current func decls
		astutil.Apply(pkg, func(c *astutil.Cursor) bool {
			tc := *c
			this.cursors[c.Node()] = &tc
			switch te := c.Node().(type) {
			case *ast.TypeSpec:
				// log.Println("typedef", t.Name.Name)
				this.typeDeclsm[te.Name.Name] = te
			case *ast.FuncDecl:
				if te.Recv != nil && te.Recv.NumFields() > 0 {
					varty := te.Recv.List[0].Type
					if ve, ok := varty.(*ast.StarExpr); ok {
						varty2 := ve.X
						tyname := varty2.(*ast.Ident).Name
						fnfullname := tyname + "_" + te.Name.Name
						this.funcDeclsm[fnfullname] = te
						curfds = append(curfds, fnfullname)
					} else if ve, ok := varty.(*ast.Ident); ok {
						tyname := ve.Name
						fnfullname := tyname + "_" + te.Name.Name
						this.funcDeclsm[fnfullname] = te
						curfds = append(curfds, fnfullname)
					} else {
						log.Println("todo", varty, reflect.TypeOf(te.Recv.List[0]))
					}
				} else {
					this.funcDeclsm[te.Name.Name] = te
					curfds = append(curfds, te.Name.Name)
				}
			case *ast.CallExpr:
				if len(curfds) == 0 { // global scope call
					switch be := te.Fun.(type) {
					case *ast.SelectorExpr:
						if iscsel(be.X) {
							break
						} else {
							log.Println("wtf", te, te.Fun, reflect.TypeOf(te.Fun))
						}
					default:
						log.Println("wtf", te, te.Fun, reflect.TypeOf(te.Fun))
					}
					// break
				} else {
					var curfd = curfds[len(curfds)-1]
					switch be := te.Fun.(type) {
					case *ast.Ident:
						this.putFuncCallDependcy(curfd, be.Name)
					case *ast.SelectorExpr:
						if iscsel(be.X) {
							break
						}
						varty := this.info.TypeOf(be.X)
						tyname := sign2rety(varty.String())
						tyname = strings.TrimRight(tyname, "*")
						fnfullname := tyname + "_" + be.Sel.Name
						this.putFuncCallDependcy(curfd, fnfullname)
					default:
						log.Println("todo", te.Fun, reflect.TypeOf(te.Fun))
					}
				}
			case *ast.Ident: // func name referenced
				if len(curfds) == 0 {
					break
				}
				var curfd = curfds[len(curfds)-1]
				varobj := this.info.ObjectOf(te)
				switch varobj.(type) {
				case *types.Func:
					this.putFuncCallDependcy(curfd, te.Name)
				}
			}
			return true
		}, func(c *astutil.Cursor) bool {
			switch te := c.Node().(type) {
			case *ast.FuncDecl:
				if te.Recv != nil && te.Recv.NumFields() > 0 {
					curfds = curfds[:len(curfds)-1]
				} else {
					curfds = curfds[:len(curfds)-1]
				}
			default:
				gopp.G_USED(te)
			}
			return true
		})
	}
}

func (pc *ParserContext) putTyperefDependcy(funame, tyname string) {

}

func (pc *ParserContext) putFuncCallDependcy(name0, name1 string) {
	if name0 == name1 {
		return
	}
	if _, ok := builtinfns[name1]; ok {
		return
	}
	n0, ok0 := pc.funcdeclNodes[name0]
	if !ok0 {
		n0 = pc.gb.MakeNode()
		*n0.Value = name0
		pc.funcdeclNodes[name0] = n0
	}
	n1, ok1 := pc.funcdeclNodes[name1]
	if !ok1 {
		n1 = pc.gb.MakeNode()
		*n1.Value = name1
		pc.funcdeclNodes[name1] = n1
	}
	// log.Println("adding", name0, n0.Value, "->", name1, n1.Value)
	err := pc.gb.MakeEdge(n1, n0)
	gopp.ErrPrint(err, name0, name1)
}
