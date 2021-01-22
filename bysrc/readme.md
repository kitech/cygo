
### Install

    go build -o cygo

### Run 

    ./bysrc

### TODO
* [ ] type assertion
* [ ] reflect
* [ ] dynamic stack size
* [ ] copy stack
* [ ] test code transpile to C???
* [ ] defer in loop
* [ ] allocator
* [x] xbuiltin, use go syntax implement some functions
* [ ] improve compile speed
* [ ] function like macro direct call

### C 符号类型自动推导
* [ ] xlab/c-for-go + modernc.org/cc 方案，测试已经完成
* [x] 准备弃用。 使用 tcc + tree-sitter做自动C头文件解析，C符号类型推导，支持函数返回值，结构体（带字段）模拟，全局变量，#define的常量，enum常量。
* [ ] 备用方案，go-clang。
* [x] 已弃用。 对于调用C函数的返回值类型， 一种typeof()，一种是使用C++的decltype()，一种手动声明函数，只需要返回值类型，像V语言中的实现一样。另外，还要知道一个 C.symbol是一个函数，还是变量，还是常量，还是类型。
以及在基本类型的操作之后的类型，像 char** p; p[0] 的类型还推导不出来。
所以不必一定要调用cgo工具来检测函数类型。
效果还不错。

### 关于语法
* [x] struct 声明语法，去掉 type
* [ ] 数组的声明，如果有初始化元素，则去掉类型部分. ints := [1, 2, 3]
* [ ] map的声明，如果有初始化元素，则去掉类型部分. ints := {"a":1, "b":2, "c":3}
* [x] varidict parameters
* [ ] 全引用的方式，不要出现指针类型
* [x] 需要定义几个常用的内置类型， unsafe.Pointer => voidptr, byteptr
* [ ] usize/isize类型
* [x] len(), cap()的写法使用方法方式
* [x] assert, sizeof, alignof, offsetof 内置
* [ ] typeof
* [+] C union 支持。转到go要用 struct
* [x] := range => in
* [x] for 循环中占位符号省略，对slice和map都可以省略两个占位符
* [x] for x in low..high
* [ ] 给 for 添加 Index项，即 slice, 则使用 Index, Value, 是map, 则，Index, Key, Value 
* [ ] for 中 index 的位置想放在后面了，但是多数语言放前面
* [x] 内置的string要有更多的方法，用单独的strings包有点麻烦
* [x] 内置的int/float要有更多的转换方法
* [ ] 结构体的通用 repr 函数
* [x] string和array/hashmap似乎可以全在builtin中用go语法实现了。
* [x] func type parameters
* [ ] type order graph
* [x] catch 语法错误/异常处理
* [ ] let 替换const
* [x] 三元运算符号，值表达式只准使用常量或者ident。
      使用内置函数 ifelse 实现类似的功能
* [ ] 生成 enum 字符串名字
* [ ] 如果没有return，则返回默认值
* [ ] integer 类型的 minval()/maxval()方法，取各类型的最大值最小值 
* [ ] 结构体setfinalizer/setdtor方法
* [ ] 在结构体声明的时候，可以给成员赋初值,类py,rb。机制，默认初始化为0,如果为nil,则不初始化
* [ ] 目标，动态性良好的静态编译语言
* [ ] print时自动解 tuple 不错
* [ ] #include 支持, 提升C兼容级别，不需要包含在 import "C"块中
* [ ] pprof支持

### 语法树重写
* [x] 变量声明解构为每条语句一个变量，如果有多个的话
* [x] 末尾没有return的要补上
* [x] 现在的ast在插入删除语句上有点弱
      实现了一种transform，可以添加语句，保持正确语序
      一般不删除语句，采用replace语句或者表达式的方式实现，效果还不错
* [ ] 使用 tuple 的地方，是否可以通过重写语法树，让编译器更容易处理
* [ ] 需要优化了, ast遍历次数过多。每一个表达式赋值给一个临时变量，似乎可解
* [ ] ast上下文依赖怎么处理
* [ ] 临时变量添加优化

### 类似项目
* https://github.com/DQNEO/minigo

### BUGS
* crash: Collecting from unknown thread
  curl thread based DNS resolve

### Depends
* go
* gcc /clang编译最终结果
* lab/c-for-go 和 modernc.org/cc
* tcc C语言宏预处理 gcc -E
* [-] tree-sitter 解析C符号类型
* vendor/{go,internal} 来自 go 官方编译器源码，有些改动

### 目标语言 
* [x] C
* [ ] javascript/wasm
* [ ] llvmir https://godoc.org/llvm.org/llvm/bindings/go https://github.com/llir/llvm pure Go

### Go AST/Compiler/Internal
* https://eli.thegreenplace.net/2019/go-compiler-internals-adding-a-new-statement-to-go-part-1/

