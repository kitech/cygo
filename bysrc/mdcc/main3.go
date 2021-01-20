package main

import (
	"gopp"
	"log"
	"reflect"

	// https://github.com/xlab/c-for-go/commit/9a426bcc5e562dfa41d7d122b6c5777b8f84d8f8
	"github.com/xlab/c-for-go/parser"
	"github.com/xlab/c-for-go/translator"
	"modernc.org/cc"
)

// demo modernc.org/cc header parser
func main() {
	cfg := &parser.Config{}
	cfg.IncludePaths = []string{"/usr/include",
		"/usr/lib/gcc/x86_64-pc-linux-gnu/10.2.0/include"}
	cfg.SourcesPaths = []string{"/usr/include/stdio.h"}
	log.Println(cfg)
	toptu, err := parser.ParseWith(cfg)
	gopp.ErrPrint(err)
	//log.Println(toptu)

	idents := toptu.Declarations.Identifiers
	for idx, ident := range idents {
		dd := ident.Node.(*cc.DirectDeclarator)
		if false {
			log.Printf("%d %#v %v %#v\n",
				idx, ident, reflect.TypeOf(ident.Node), dd)
		}
	}
	tu := toptu
	cnt := 0
	for tu != nil {
		extdecl := tu.ExternalDeclaration
		fndef := extdecl.FunctionDefinition
		//log.Println(cnt, extdecl.Case, extdecl.FunctionDefinition)
		switch extdecl.Case {
		case 0:
			// FunctionDefinition
			typ := fndef.Declarator.Type
			nameb := fndef.Declarator.DirectDeclarator.DirectDeclarator.Token.S()
			name := string(nameb)
			log.Println(cnt, typ, name)
		case 1:
			// Declaration
			d := extdecl.Declaration
			// typ := d.InitDeclaratorListOpt.InitDeclaratorList.InitDeclarator.Declarator.Type
			dlstopt := d.InitDeclaratorListOpt
			if dlstopt == nil {
				log.Println(cnt, extdecl.Case, "wtt", d)
			} else {
				typ := dlstopt.InitDeclaratorList.InitDeclarator.Declarator.Type
				nameb := dlstopt.InitDeclaratorList.InitDeclarator.Declarator.DirectDeclarator.Token.S()
				name := string(nameb)
				log.Println(cnt, extdecl.Case, typ, name, dlstopt.InitDeclaratorList.InitDeclarator.Declarator.DirectDeclarator.DirectDeclarator)
			}
		}
		cnt++
		tu = tu.TranslationUnit
	}

	tlcfg := &translator.Config{}
	tl, err := translator.New(tlcfg)
	gopp.ErrPrint(err)
	tl.Learn(toptu)
	// log.Println(tl)
	log.Println(tl.Declares())
	for idx, decl := range tl.Declares() {
		log.Println(idx, decl.Spec.Kind(), decl.Name, reflect.TypeOf(decl.Spec))
		switch spec := decl.Spec.(type) {
		case *translator.CFunctionSpec:
			var retkd translator.CTypeKind
			if spec.Return != nil {
				retkd = spec.Return.Kind()
			}
			var gotyspec translator.GoTypeSpec
			if spec.Return != nil {
				gotyspec = tl.TranslateSpec(spec.Return)
			}
			log.Println(idx, decl.Spec.Kind(), decl.Name, spec.Return, retkd,
				reflect.TypeOf(spec.Return), gotyspec)
		}
	}

}
