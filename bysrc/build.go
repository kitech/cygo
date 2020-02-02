package main

// 总体，取代main只的循环不同包的逻辑

// 纵向分步，
// 1 解析出来所有的包，找到c code, c symbol, 但不做类型check
// 2 解析 c code, 为 c symbol 生成全局fakec 包
// 3 做类型check, 语义检查
// 4 生成最终代码

// 第2步能够节省很多时间

type builder struct {
	// pkgs/funcs/types depgraph
}
