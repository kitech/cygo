all:

prs:
	#antlr4 -long-messages
	mkdir -p parser
	# cp with sed modify
	sed 's/package GoParseTree/package parser/' Go/go_parser_base.go > parser/go_parser_base.go
	antlr4 -long-messages -Dlanguage=Go -o parser GoLexer.g4
	antlr4 -long-messages -Dlanguage=Go -o parser GoParser.g4
	go install -v ./parser/

cpl:
	go build -v

