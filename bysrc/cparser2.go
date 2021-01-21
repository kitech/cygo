package main

// try parse C use modernc.org/cc

import (
	"fmt"
	"go/token"
	"go/types"
	"gopp"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"reflect"
	"runtime"
	"strings"

	cc1p "github.com/xlab/c-for-go/parser"
	cc1t "github.com/xlab/c-for-go/translator"
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

		if err == nil {
			cp.collects1()
			// cp.dumpdecls1()
		}

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
			tlcfg := &cc1t.Config{}
			tl, err := cc1t.New(tlcfg)
			gopp.ErrPrint(err)
			tl.Learn(ctu)
			cp.cctr = tl
			//log.Println(tl)
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
		return "string", types.Typ[types.String]
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
		log.Println("in TagMap")
	}
	if _, ok := cp.cctr.ValueMap()[sym]; ok {
		log.Println("in ValueMap")
	}
	if _, ok := cp.cctr.ExpressionMap()[sym]; ok {
		log.Println("in ExpressionMap")
	}
	for idx, v := range cp.cctr.Defines() {
		log.Println(idx, v)
	}

	log.Panicln("not found???", sym)
	typ := types.Typ[types.String]
	return "", typ
}
func (cp *cparser2) gotypeof(sym string) types.Type {
	if strings.HasPrefix(sym, "struct_") {
		old := sym
		sym = sym[7:]
		log.Println("mapsymto", old, sym)
	}

	csi, found := cp.syms[sym]
	if found {
		if csi.kind == csym_define {
			return types.Typ[types.UntypedInt]
		}

		// TODO symbol kind
		return cp.gotypeof2(sym, csi.typ, true)
	} else {
		syms := map[string]types.Type{
			"__FILE__": types.Typ[types.String],
			"__LINE__": types.Typ[types.Int],
		}
		if tyobj, ok := syms[sym]; ok {
			return tyobj
		}

		// primitive_type
		amdl := cp.ctu1.Model
		vmdl := reflect.ValueOf(amdl).Elem()
		for tk, _ := range amdl.Items {
			if sym == strings.ToLower(tk.String()) ||
				sym == tk.CString() {
				tyfname := fmt.Sprintf("%sType", tk.String())
				fval := vmdl.FieldByName(tyfname)
				fval2 := fval.Interface().(cc1.Type)
				// log.Println(fval, reftyof(fval2), fval2.String(), fval2.Kind())
				return cp.gotypeof2(sym, fval2, true)
			}
		}
		// log.Fatalln("symbol not found", sym)
	}
	return nil
}

func (cp *cparser2) gotypeof2(sym string, cty cc1.Type, resty bool) types.Type {
	switch cty.Kind() {
	case cc1.Struct:
		fields, iscomplete := cty.Members()
		if !iscomplete {
			log.Println(cty, iscomplete, len(fields), cty.Declarator().Type)
		}
		tyn1 := types.NewTypeName(token.NoPos, fcpkg, cty.String(), nil)
		named1 := types.NewNamed(tyn1, nil, nil)
		var fldvars []*types.Var
		for idx, fldx := range fields {
			fldtyx := fldx.Type
			iscyclerefty2 := fldtyx.String() == cty.String() ||
				fldtyx.String() == cty.Pointer().String()
			var fldty types.Type
			if iscyclerefty2 {
				fldty = named1
			} else {
				if fldtyx.Kind() == cc1.Array {
					fldty = cp.gotypeof2(sym, fldtyx, resty)
				} else {
					fldty = cp.gotypeof2(sym, fldtyx, resty)
				}
			}

			tkval, bindings := fldtyx.Declarator().Identifier()
			gopp.Assert(tkval == fldx.Name, "wtfff", bindings)
			fldidtx := bindings.Lookup(cc1.NSIdentifiers, tkval)
			dirdecl := fldidtx.Node.(*cc1.DirectDeclarator)
			fldname := cp2_token_toname(dirdecl.Token.String())
			log.Println(idx, fldx.Name, fldtyx.Kind(), fldtyx,
				iscyclerefty2, fldname, fldty, cty.Pointer())
		}
		st1 := types.NewStruct(fldvars, nil)
		named1.SetUnderlying(st1)
		return named1
	case cc1.Union: // TODO
		log.Panicln(sym)
	case cc1.Array:
		ety := cty.Element()
		goety := cp.gotypeof2(sym, ety, resty)
		if cty.Elements() == 0 {
			goty := types.NewSlice(goety)
			return goty
		} else {
			goty := types.NewArray(goety, int64(cty.Elements()))
			return goty
		}
	case cc1.Enum:
		return types.Typ[types.Int]
	case cc1.Function:
		ty := cty.Result()
		// log.Println(cp.name, sym, cty.Kind(), ty.Kind(), ty)
		return cp.gotypeof2(sym, ty, resty)
	case cc1.Void:
		return types.NewTuple()
	case cc1.UintPtr:
		return types.Typ[types.Uintptr]
	case cc1.Ptr:
		switch tk2 := cty.Element().Kind(); tk2 {
		case cc1.Void:
			return types.Typ[types.Voidptr]
		case cc1.Char, cc1.SChar, cc1.UChar:
			return types.Typ[types.Byteptr]
		}
		undty := cp.gotypeof2(sym, cty.Element(), resty)
		return types.NewPointer(undty)
	case cc1.Char, cc1.SChar:
		return types.Typ[types.Int8]
	case cc1.UChar:
		return types.Typ[types.Byte]
	case cc1.Bool:
		return types.Typ[types.Bool]
	case cc1.ULong, cc1.ULongLong:
		return types.Typ[types.Uint64]
	case cc1.Long, cc1.LongLong:
		return types.Typ[types.Int64]
	case cc1.UInt:
		return types.Typ[types.Uint]
	case cc1.Int:
		return types.Typ[types.Int]
	case cc1.UShort:
		return types.Typ[types.Uint16]
	case cc1.Short:
		return types.Typ[types.Int16]
	case cc1.Double:
		return types.Typ[types.Float64]
	case cc1.FLOAT:
		return types.Typ[types.Float32]
	default:
		log.Println(cp.name, sym, cty.Kind(), cty)
	}
	return nil
}

// ddl.Token format: 3 parts: file id name
func cp2_token_toname(tkstr string) string {
	idname := strings.Trim(strings.Split(tkstr, " ")[2], "\"")
	return idname
}

func (cp *cparser2) collects1() {
	ctu1 := cp.ctu1
	declids1 := ctu1.Declarations.Identifiers
	decltags1 := ctu1.Declarations.Tags // struct/union/enum here
	for ikey, ido := range declids1 {
		_ = ikey
		ddl := ido.Node.(*cc1.DirectDeclarator)
		td := ddl.TopDeclarator()
		// log.Println(ikey, td.Type, "//", ddl.EnumVal, td.Type.Kind(),
		//	ddl.Token, "//", ddl.Token2, "//", ddl.Token3)
		// ddl.Token format: 3 parts: file id name
		idname := strings.Trim(strings.Split(ddl.Token.String(), " ")[2], "\"")
		cp.syms[idname] = newcsymdata2(idname, int(td.Type.Kind()), td.Type)
	}
	for ikey, tagx := range decltags1 {
		_ = ikey
		switch tago := tagx.Node.(type) {
		case *cc1.StructOrUnionSpecifier:
			sty := tago.Declarator().Type
			// members, _ := sty.Members()
			stname := strings.Split(sty.String(), " ")[1]
			// log.Println(ikey, sty.Kind(), stname, sty, "fieldcnt", len(members))
			cp.syms[stname] = newcsymdata2(stname, csym_struct, sty)
		case *cc1.EnumSpecifier:
			// log.Println(ikey, "enum", tago.EnumeratorList.Enumerator.Value)
		case xc.Token:
			// log.Println(ikey, "xc.Token?", tago.Val)
		default:
			log.Println(ikey, reftyof(tagx.Node))
		}
	}
	for ikey, macx := range ctu1.Macros {
		_ = ikey
		idname := strings.Trim(strings.Split(macx.DefTok.String(), " ")[2], "\"")
		// log.Println(ikey, idname, macx.Type, macx.Value)
		cp.syms[idname] = newcsymdata2(idname, csym_define, nil)
	}

	log.Println("macros", len(ctu1.Macros), "syms", len(cp.syms))
}

func (cp *cparser2) dumpdecls1() {
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
	code = c11_builtin_atomic_defs + codepfx + code
	filename, err := cprsavetmp(cp.name, code)
	err = cp.parsefile(filename)
	return err
}

// preprocessor
func (cp *cparser2) cpp() {}

// parser
// check
