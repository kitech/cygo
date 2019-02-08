module github.com/pwaller/go2ll

go 1.12

require (
	github.com/google/pprof v0.0.0-20190109223431-e84dfd68c163 // indirect
	github.com/ianlancetaylor/demangle v0.0.0-20181102032728-5e5cf60278f6 // indirect
	github.com/llir/llvm v0.3.0-pre6
	golang.org/x/arch v0.0.0-20181203225421-5a4828bb7045 // indirect
	golang.org/x/crypto v0.0.0-20190123085648-057139ce5d2b // indirect
	golang.org/x/sys v0.0.0-20190124100055-b90733256f2e // indirect
	golang.org/x/tools v0.0.0-20190125232054-d66bd3c5d5a6
)

replace github.com/llir/llvm => ./llvm
