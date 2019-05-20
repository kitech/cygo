package main

import (
	"gopp"
	"io/ioutil"
	"log"
	"os"
)

var fname string

func main() {
	if len(os.Args) < 2 {
		log.Fatalln("must specify a go source file to tranpiler")
	}
	fname = os.Args[1]

	psctx := NewParserContext(fname)
	err := psctx.Init()
	gopp.ErrPrint(err)

	g2n := g2nim{}
	g2n.psctx = psctx
	g2n.genpkgs()

	code := "import os, threadpool\n\n"
	code += "include \"nrtbase.nim\"\n\n"
	code += g2n.code()
	code += "\n\n"
	code += "main()"
	code += "\n\n"

	ioutil.WriteFile("opkgs/foo.nim", []byte(code), 0644)
}
