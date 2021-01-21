package main

// try parse C use modernc.org/cc

import (
	"go/types"
	"gopp"
	"io/ioutil"
	"log"
	"os"
	"reflect"
	"runtime"
	"strings"

	"github.com/xlab/c-for-go/generator"
	cc1p "github.com/xlab/c-for-go/parser"
	cc1t "github.com/xlab/c-for-go/translator"
	"gopkg.in/yaml.v2"
	cc1 "modernc.org/cc"

	// cc2 "modernc.org/cc/v2"
	cc3 "modernc.org/cc/v3"
	"modernc.org/xc"
)

// 突然发现不支持 C11 atomic
// 还不支持__thread变量？
type cparser2 struct {
	name    string
	predefs string // like -DGC_THREADS
	cctr    *cc1t.Translator
	cfg     *cc3.Config
	//ctu2    *cc2.TranslationUnit
	ctu1 *cc1.TranslationUnit
	ctu3 *cc3.AST // not support

	syms map[string]*csymdata2 // identity/struct/type name =>

}
type csymdata2 struct {
	name string
	kind int
	typ  cc1.Type
}

func newcsymdata2(name string, kind int, typ cc1.Type) *csymdata2 {
	csi := &csymdata2{}
	csi.name = name
	csi.kind = kind
	csi.typ = typ
	return csi
}

func newcparser2(name string) *cparser2 {
	cp := &cparser2{}
	cp.name = name
	cp.syms = map[string]*csymdata2{}
	return cp
}

func newfilesource(filename string) *cc3.Source {
	srco := &cc3.Source{}
	srco.Name = filename
	return srco
}
func newstrsource(code string) *cc3.Source {
	srco := &cc3.Source{}
	srco.Name = "flycode"
	srco.Value = code
	return srco
}

// for modernc.org/cc
// https://github.com/gcc-mirror/gcc/blob/master/gcc/memmodel.h
var c11_builtin_atomic_defs = `
#define __ATOMIC_RELAXED  0
#define __ATOMIC_CONSUME  1
#define __ATOMIC_ACQUIRE  2
#define __ATOMIC_RELEASE  3
#define __ATOMIC_ACQ_REL  4
#define __ATOMIC_SEQ_CST  5
#define __ATOMIC_LAST  6
`

// add to xlab/c-for-go/parser/predefined.go
var extra_fix_ccpredefs = `
#define __thread
#define __builtin_va_start(x, y)
#define __builtin_va_end(x)
#define voidptr void*
#define byteptr char*
#define byte char
#define bool char
#define nilptr ((void*)0)
`

func (cp *cparser2) ccHostConfig() (
	predefsm map[string]interface{}, incpaths, sysincs []string, err error) {
	var predefs string

	os.Setenv("LANG", "C")
	os.Setenv("LC_ALL", "C")
	os.Setenv("LC_CTYPE", "C")

	// predefs format: #define Foo 123\n
	predefs, incpaths, sysincs, err = cc1.HostConfig()
	gopp.ErrFatal(err, cp.name, "can ignore")
	if err != nil {
		if false {
			predefs, incpaths, sysincs, err = cc3.HostConfig("")
			gopp.ErrPrint(err, cp.name, "can ignore")

			if err != nil {
				incpaths = append(incpaths, preincdirs...)
				sysincs = append(sysincs, presysincs...)
				err = nil
			}
		}
	}
	os.Unsetenv("LANG")
	os.Unsetenv("LC_ALL")
	os.Unsetenv("LC_CTYPE")
	//log.Fatalln(predefs, incpaths, sysincs)

	predefs2 := " -D__ATOMIC_RELAXED=0 -D__ATOMIC_CONSUME=1 -D__ATOMIC_ACQUIRE=2 -D__ATOMIC_RELEASE=3 -D__ATOMIC_ACQ_REL=4 -D__ATOMIC_SEQ_CST=5 "
	predefs2 += " " + cp.predefs + " -DGC_THREADS"
	predefsm = cp2_split_predefs(predefs2)
	//predefsm = map[string]interface{}{}
	// predefsm["__thread"] = ""

	pwdir, err := os.Getwd()
	if pwdir != "" {
		incpaths = append(incpaths, pwdir)
	}
	log.Println(len(strings.Split(predefs, "\n")), incpaths, sysincs)
	return
}

func (cp *cparser2) parsefile(filename string) error {
	predefs, incpaths, sysincs, err := cp.ccHostConfig()
	cfg := &cc3.Config{}
	cfg.ABI, err = cc3.NewABI(runtime.GOOS, runtime.GOARCH)
	gopp.ErrPrint(err)
	// log.Println(cfg.ABI)

	if false {
		// not work for bits/types.h
		srco := newfilesource(filename)
		cfg.RejectIncludeNext = false
		ctu, err := cc3.Parse(cfg, incpaths, sysincs, []cc3.Source{*srco})
		// ctu, err := cc.Translate(cfg, incpaths, sysincs, []cc.Source{*srco})
		gopp.ErrPrint(err, ctu != nil, filename)
		cp.ctu3 = ctu
		os.Exit(-1)
	}
	if false {
		// not work for stdarg.h
		/*
			srco, err := cc2.NewFileSource(filename)
			gopp.ErrPrint(err, filename)
			cfg := &cc2.Tweaks{}
			cfg.EnableImplicitBuiltins = true
			cfg.EnableImplicitDeclarations = true
			ctu, err := cc2.Translate(cfg, incpaths, sysincs, srco)
			gopp.ErrPrint(err)
			cp.ctu2 = ctu
			os.Exit(-1)
		*/
	}
	if false {
		// not work for c11 stdatomic.h
		paths := append(incpaths, sysincs...)
		cfg := &cc1p.Config{}
		cfg.IncludePaths = paths
		cfg.SourcesPaths = []string{filename}
		cfg.Defines = predefs
		// cfg.CCDefs = true
		// cfg.CCIncl = true
		// patch cc1x:100: 	model := *models[cfg.archBits]
		// patch cc1x:106:  // cc.EnableIncludeNext(),
		ctu, err1 := cc1p.ParseWith(cfg)
		err = err1
		gopp.ErrPrint(err, filename)
		cp.ctu1 = ctu

	}

	if true {
		paths := append(incpaths, sysincs...)
		cfg := &cc1p.Config{}
		cfg.IncludePaths = paths
		cfg.SourcesPaths = []string{filename}
		cfg.Defines = predefs

		ctu, err1 := cc1p.ParseWith(cfg)
		err = err1
		gopp.ErrPrint(err, filename)
		cp.ctu1 = ctu

		if err == nil {
			configPath := "expall.yml"
			cfgData, err := ioutil.ReadFile(configPath)
			if err != nil {
				return err
			}
			type ProcessConfig struct {
				Generator  *generator.Config `yaml:"GENERATOR"`
				Translator *cc1t.Config      `yaml:"TRANSLATOR"`
				Parser     *cc1p.Config      `yaml:"PARSER"`
			}
			var cfg ProcessConfig
			if err := yaml.Unmarshal(cfgData, &cfg); err != nil {
				return err
			}

			tlcfg := &cc1t.Config{}
			tlcfg = cfg.Translator
			//tlcfg.ConstRules = cc1t.ConstRules{"defines": "expand", "enums": "expand"}
			tl, err := cc1t.New(tlcfg)
			gopp.ErrPrint(err)
			tl.Learn(ctu)
			cp.cctr = tl
			//log.Println(tl)
			// tl.Defines() filtered by rules
			//log.Println(tl.Defines())
			//log.Println(tl.Typedefs())
			//log.Println(tl.Declares())
			//log.Fatalln("===", "defines", len(tl.Defines()), "typedefs", len(tl.Typedefs()), "declares", len(tl.Declares()), cfg.Translator)
		}
	}
	if true {
		// cc/v3
	}
	return err
}

func trtypespec2gotypes(trtyp cc1t.GoTypeSpec) types.Type {
	log.Printf("%s %#v %v %v\n", trtyp.String(), trtyp, trtyp.Kind, "=>...")
	switch trtyp.String() {
	case "[]byte":
		//typ := types.Typ[types.Byteptr]
		return types.Typ[types.Byteptr]
	case "[][]byte":
		udtyp := types.Typ[types.Byteptr]
		typ := types.NewPointer(udtyp)
		//log.Println(trtyp, udtyp, typ)
		return typ
	case "unsafe.Pointer":
		typ := types.Typ[types.Voidptr]
		return typ
	case "float64":
		typ := types.Typ[types.Float64]
		return typ
	case "int32":
		typ := types.Typ[types.Int32]
		return typ
	case "uint32":
		typ := types.Typ[types.Uint32]
		return typ
	case "time_t", "size_t", "__pid_t":
		typ := types.Typ[types.Usize]
		return typ
	default:
		log.Panicln("noimpl", trtyp)
	}
	return types.Typ[types.Int]
}

func cctype2gotypes(typ cc1.Type) types.Type {
	switch typ.Kind() {
	case cc1.Int:
		typ := types.Typ[types.Int]
		return typ
	default:
		log.Panicln("noimpl", typ)
	}
	return types.Typ[types.Int]
}

func (cp *cparser2) symtype(sym string) (string, types.Type) {
	switch sym {
	case "__FILE__", "__FUNCTION__":
		return "string", types.Typ[types.Byteptr]
	case "__LINE__", "errno":
		return "int", types.Typ[types.Int]
	}

	//log.Println(cp.cctr.Declares())
	if cp.cctr == nil {
		log.Panicln("wtt", sym)
	}
	for k, v := range cp.cctr.Declares() {
		if sym == v.Name {
			log.Println(k, v.Name, reflect.TypeOf(v.Spec))
			switch spec := v.Spec.(type) {
			case *cc1t.CFunctionSpec:
				if spec.Return == nil {
					// void??? => int
					return types.Voidty.String(), types.Voidty
					//return "int", types.Typ[types.Int]
				}
				trtyp := cp.cctr.TranslateSpec(spec.Return)
				dsty := trtypespec2gotypes(trtyp)
				log.Printf("%#v %v\n", spec, dsty)
				return dsty.String(), dsty
			case *cc1t.CTypeSpec:
				trtyp := cp.cctr.TranslateSpec(spec)
				dsty := trtypespec2gotypes(trtyp)
				log.Printf("%#v %v\n", spec, dsty)
				return dsty.String(), dsty
			}
			log.Panicln("got", sym)
		}
	}
	log.Println(cp.cctr.Defines())
	log.Println(cp.cctr.Typedefs())
	// log.Println(cp.ctu1)
	//log.Println(cp.cctr.TagMap())
	for id, macro := range cp.ctu1.Macros {
		name := string(xc.Dict.S(macro.DefTok.Val))
		if name == sym {
			if macro.Type == nil {
				log.Println(id, macro, "/", macro.Type)
				return "int", types.Typ[types.Int]
			}
			log.Println(id, macro, "/", macro.DefTok.Val, string(xc.Dict.S(macro.DefTok.Val)),
				macro.Type.Kind(), reflect.TypeOf(macro.Type))
			dsty := cctype2gotypes(macro.Type)
			log.Println(sym, dsty)
			return dsty.String(), dsty
			//break
		}
	}
	if _, ok := cp.cctr.TagMap()[sym]; ok {
		log.Println("symin TagMap")
	}
	if _, ok := cp.cctr.ValueMap()[sym]; ok {
		log.Println("symin ValueMap")
	}
	if _, ok := cp.cctr.ExpressionMap()[sym]; ok {
		log.Println("symin ExpressionMap")
	}
	for idx, v := range cp.cctr.Defines() {
		log.Println(idx, v)
	}

	log.Panicln("not found???", sym)
	typ := types.Typ[types.String]
	return "", typ
}

// preprocessor
func (cp *cparser2) cpp() {}

// parser
// check
