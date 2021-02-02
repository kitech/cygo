package main

import (
	"fmt"
	"gopp"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/thoas/go-funk"
)

// 总体，取代main只的循环不同包的逻辑

// 纵向分步，
// 1 解析出来所有的包，找到c code, c symbol, 但不做类型check
// 2 解析 c code, 为 c symbol 生成全局fakec 包
// 3 做类型check, 语义检查
// 4 生成最终代码

// 第2步能够节省很多时间
// 还可以考虑处理编译flags的问题

type builder struct {
	// pkgs/funcs/types depgraph
}

type cbuilder struct {
	filepath string
	cflags   []string
	ldflags  []string
}

func newcbuilder() *cbuilder {
	this := &cbuilder{}
	return this
}

func (this *cbuilder) setsrc(filepath string) {
	this.filepath = filepath
}
func (this *cbuilder) embedccode(code string) {
	log.Println(len(code), code)
	lines := strings.Split(code, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "#cgo ") {
			this.cgoline(line)
		}
	}
}
func (this *cbuilder) cgoline(line string) {
	log.Println(line)
	line = strings.TrimSpace(line)
	fields := strings.Fields(line)
	arr := []string{}
	switch fields[1] {
	case "CFLAGS:":
		for i := 2; i < len(fields); i++ {
			item := fields[i]
			if !funk.Contains(this.cflags, item) {
				arr = append(arr, item)
			}
		}
		this.cflags = append(arr, this.cflags...)
	case "LDFLAGS:":
		for i := 2; i < len(fields); i++ {
			item := fields[i]
			if !funk.Contains(this.ldflags, item) {
				arr = append(arr, item)
			}
		}
		this.ldflags = append(arr, this.ldflags...)
	default:
		log.Panicln("wtt", line)
	}
}

func (this *cbuilder) build() {
	os.Setenv("LC_ALL", "C")
	exe := "gcc"
	args := []string{"-g", "-O0", "-fPIC", "-std=gnu99"}
	args = append(args, this.cflags...)
	args = append(args, this.ldflags...)
	args = append(args, "./opkgs/foo.c")

	fmt.Println("===>", exe, args)
	cmdo := exec.Command(exe, args...)
	if true {
		btime := time.Now()
		output, err := cmdo.CombinedOutput()
		gopp.ErrPrint(err, args, len(output))
		log.Println(string(output))
		if err == nil {
			fmt.Println("<===", "success", args, time.Since(btime),
				humanize.Bytes(uint64(gopp.FileSize("a.out"))))
		} else {
			gopp.ErrPrint(err, args, len(output), time.Since(btime))
		}
	}
}
