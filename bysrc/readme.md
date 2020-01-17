
### TODO
* [ ] type assertion
* [ ] reflect
* [ ] dynamic stack size

### 关于C类型
对于调用C函数的返回值类型， 一种typeof()，一种是使用C++的decltype()，一种手动声明函数，只需要返回值类型，像V语言中的实现一样。另外，还要知道一个 C.symbol是一个函数，还是变量，还是常量，还是类型。
所以不必一定要调用cgo工具来检测函数类型。

### 关于语法
* struct 声明语法，去掉 type
* 数组的声明，如果有初始化元素，则去掉类型部分. ints := [1, 2, 3]
* varidict parameters
* 
