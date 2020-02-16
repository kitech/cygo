
### TODO
* [ ] type assertion
* [ ] reflect
* [ ] dynamic stack size
* [ ] test code transpile to C???
* [ ] defer in loop
* [x] xbuiltin, use go syntax implement some function

### 关于C类型
对于调用C函数的返回值类型， 一种typeof()，一种是使用C++的decltype()，一种手动声明函数，只需要返回值类型，像V语言中的实现一样。另外，还要知道一个 C.symbol是一个函数，还是变量，还是常量，还是类型。
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
* [x] len(), cap()的写法使用方法方式
* [x] assert, sizeof, alignof, offsetof 内置
* [+] C union 支持。转到go要用 struct
* [x] := range => in
* [x] for 循环中占位符号省略，对slice和map都可以省略两个占位符
* [x] for x in low..high
* [ ] 给 for 添加 Index项，即 slice, 则使用 Index, Value, 是map, 则，Index, Key, Value 
* [ ] for 中 index 的位置想放在后面了，但是多数语言放前面
* [x] 内置的string要有更多的方法，用单独的strings包有点麻烦
* [x] 内置的int/float要有更多的转换方法
* [ ] 结构体的通用 repr 函数
* [ ] string和array似乎可以全在builtin中用go语法实现了。
* [ ] func type parameters
* [ ] type order graph
* [ ] catch 语法错误/异常处理
* [ ] let 替换const
* [ ] 三元运算符号，值表达式只准使用常量或者ident
* [ ] 生成 enum 名字
* [ ] 如果没有return，则返回默认值

### 语法树重写
* [ ] 变量声明解构为每条语句一个变量，如果有多个的话
* [x] 末尾没有return的要补上
* [ ] 现在的ast在插入删除语句上有点弱 
      实现了一种transform，可以添加语句，保持正确语序

### 类似项目
* https://github.com/DQNEO/minigo

### BUGS
* crash: Collecting from unknown thread
  curl thread based DNS resolve

### Deps
* go
* gcc
* tcc
* tree-sitter

