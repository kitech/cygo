package main

// try parse C use modernc.org/cc

import (
	"fmt"
	"gopp"
	"io/ioutil"
	"log"
	"math/rand"
	"runtime"

	cc1x "github.com/xlab/c-for-go/parser"
	cc1 "modernc.org/cc"
	"modernc.org/cc/v3"
)

type cparser2 struct {
	name string
	ctu  *cc.AST
	cfg  *cc.Config
	ctu1 *cc1.TranslationUnit
}

func newcparser2(name string) *cparser2 {
	cp := &cparser2{}
	cp.name = name
	return cp
}

func newfilesource(filename string) *cc.Source {
	srco := &cc.Source{}
	srco.Name = filename
	return srco
}
func newstrsource(code string) *cc.Source {
	srco := &cc.Source{}
	srco.Name = "flycode"
	srco.Value = code
	return srco
}

const codepfx = "#include <stdio.h>\n" +
	"#include <stdlib.h>\n" +
	"#include <string.h>\n" +
	"#include <errno.h>\n" +
	"#include <pthread.h>\n" +
	"#include <time.h>\n" +
	"#include <cxrtbase.h>\n" +
	"\n"

var preincdirs = []string{"/home/me/oss/src/cxrt/src",
	"/home/me/oss/src/cxrt/3rdparty/cltc/src",
	"/home/me/oss/src/cxrt/3rdparty/tcc"}
var presysincs = []string{"/usr/include", "/usr/local/include",
	"/usr/lib/gcc/x86_64-pc-linux-gnu/9.2.1/include"}

func ccHostConfig() (predefs string, incpaths, sysincs []string, err error) {
	predefs, incpaths, sysincs, err = cc.HostConfig("")
	gopp.ErrPrint(err)
	if err != nil {
		incpaths = append(incpaths, preincdirs...)
		sysincs = append(sysincs, presysincs...)
		err = nil
	}
	log.Println(predefs, incpaths, sysincs)
	return
}

func (cp *cparser2) parsefile(filename string) error {
	_, incpaths, sysincs, err := ccHostConfig()
	cfg := &cc.Config{}
	cfg.ABI, err = cc.NewABI(runtime.GOOS, runtime.GOARCH)
	gopp.ErrPrint(err)
	// log.Println(cfg.ABI)

	if false {
		srco := newfilesource(filename)
		ctu, err := cc.Parse(cfg, incpaths, sysincs, []cc.Source{*srco})
		// ctu, err := cc.Translate(cfg, incpaths, sysincs, []cc.Source{*srco})
		gopp.ErrPrint(err, ctu != nil, filename)
		cp.ctu = ctu
	}
	if true {
		paths := append(incpaths, sysincs...)
		cfg := &cc1x.Config{}
		cfg.IncludePaths = paths
		cfg.SourcesPaths = []string{filename}
		ctu, err1 := cc1x.ParseWith(cfg)
		err = err1
		gopp.ErrPrint(err, filename)
		cp.ctu1 = ctu
	}
	if err == nil {
		cp.dumpdecls()
	}
	return err
}

func (cp *cparser2) dumpdecls() {
	ctu1 := cp.ctu1
	declids1 := ctu1.Declarations.Identifiers
	decltags1 := ctu1.Declarations.Tags // struct/union/enum here
	for ikey, ido := range declids1 {
		ddl := ido.Node.(*cc1.DirectDeclarator)
		td := ddl.TopDeclarator()
		log.Println(ikey, td.Type, "//", ddl.EnumVal, td.Type.Kind(),
			ddl.Token, "//", ddl.Token2, "//", ddl.Token3)
		prms, varidic := td.Type.Parameters()
		_ = varidic
		for idx, prmo := range prms {
			log.Println(idx, prmo.Name, prmo.Type, prmo.Type.Kind(), prmo.Type.Element())
		}
	}
	for ikey, tagx := range decltags1 {
		switch tago := tagx.Node.(type) {
		case *cc1.StructOrUnionSpecifier:
			sty := tago.Declarator().Type
			members, _ := sty.Members()
			log.Println(ikey, "struct", sty, "fieldcnt", len(members))
		case *cc1.EnumSpecifier:
			log.Println(ikey, "enum", tago.EnumeratorList.Enumerator.Value)
		default:
			log.Println(ikey, reftyof(tagx.Node))
		}

	}

	log.Println("macros", len(ctu1.Macros), "tags", len(ctu1.Declarations.Tags),
		"declids", len(ctu1.Declarations.Identifiers))
}

func cprsavetmp(cpname string, code string) (string, error) {
	rmoldtccppfiles()
	filename := fmt.Sprintf("/tmp/tcctrspp.%s.%d.c", cpname, rand.Intn(10000000)+50000)
	cp1cache.ppfiles[filename] = 1
	err := ioutil.WriteFile(filename, []byte(code), 0644)
	return filename, err
}

func (cp *cparser2) parsestr(code string) error {
	code += codepfx
	filename, err := cprsavetmp(cp.name, code)
	err = cp.parsefile(filename)
	return err
}

// preprocessor
func (cp *cparser2) cpp() {}

// parser
// check
