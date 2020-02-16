package adt

type Stack struct {
	data []voidptr
}

func NewStack() *Stack {
	stk := &Stack{}
	return stk
}

func (stk *Stack) Push(v voidptr) bool {
	stk.data.append(v)
	return true
}

func (stk *Stack) Pop() voidptr {
	if stk.data.len == 0 {
		return nil
	}
	v := stk.data[stk.data.len-1]
	stk.data = stk.data[:stk.data.len-1]
	return v
}

func (stk *Stack) Empty() bool {
	return stk.data.len == 0
}
func (stk *Stack) Len() int {
	return stk.data.len
}

func (stk *Stack) Clear() {
	if stk.data.len == 0 {
		return
	}
	stk.data.clear()
}
