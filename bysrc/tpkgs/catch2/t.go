package main

func foo1() error {
	return nil
}
func foo2() (int, error) {
	foo1()
	return 0, nil
}
func main() {
	var err1 error

	foo1()
	println(111)
	foo2()
	println(222)

	catch{
		case  nil :
		case err1:
		default:
		println(err)
		if true {
			continue
		}else{
			break
		}
	}
}


