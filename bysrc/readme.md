
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
* struct 声明语法，去掉 type
* 数组的声明，如果有初始化元素，则去掉类型部分. ints := [1, 2, 3]
* map的声明，如果有初始化元素，则去掉类型部分. ints := {"a":1, "b":2, "c":3}
* varidict parameters
* 全引用的方式，不要出现指针类型
* [x] 需要定义几个常用的内置类型， unsafe.Pointer => voidptr, byteptr
* [x] len(),cap()的写法使用方法方式
* [x] assert, sizeof, alignof, offsetof 内置
* [ ] union 支持
* [ ] := range => in
* [x] 内置的string要有更多的方法，用单独的strings包有点麻烦
* [x] 内置的int/float要有更多的方法
* [ ] string和array似乎可以全在builtin中用go语法实现了。

### 语法树重写
* [ ] 变量声明解构为每条语句一个变量，如果有多个的话
* [ ] 末尾没有return的要补上

