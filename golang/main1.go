package main

import (
	"io/ioutil"
	"log"
	"time"

	"github.com/antlr/antlr4/runtime/Go/antlr"

	"cxrt/golang/parser"
)

func main() {
	fstm, err := antlr.NewFileStream("./examples/method.go")
	if err != nil {
		log.Panicln(err, fstm)
	}
	bcc, err := ioutil.ReadFile("./examples/method.go")
	if err != nil {
		log.Panicln(err)
	}

	strstm := antlr.NewInputStream(string(bcc))
	// lexer := parser.NewGoLexer(fstm)
	lexer := parser.NewGoLexer(strstm) // still slow
	tkstm := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)

	// Create the Lexer
	prs := parser.NewGoParser(tkstm)
	log.Println("prs", prs)

	btime := time.Now()
	antlr.ParseTreeWalkerDefault.Walk(&dumpgoListerner{}, prs.SourceFile())
	dtime := time.Since(btime)
	log.Println(dtime) // about 1.2s, so slow for donothing?
}

type dumpgoListerner struct {
	parser.BaseGoParserListener
}

func (l *dumpgoListerner) EnterSourceFile(c *parser.SourceFileContext) {
	log.Println(c.AllDeclaration())
}
func (l *dumpgoListerner) EnterImportDecl(c *parser.ImportDeclContext) {
	log.Println(c.GetText())
	log.Println(c.AllImportSpec())
}

func (l *dumpgoListerner) EnterMethodDecl(c *parser.MethodDeclContext) {
	log.Println(c.Receiver().GetChildCount())
}
