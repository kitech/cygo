package adt

func test_stk1() {
	stk := NewStack()
	println(stk.Len(), stk.Empty())
	stk.Push(q)
	println(stk.Len(), stk.Empty())
	stk.Clear()
	println(stk.Len(), stk.Empty())

	stk.Push(q)
	q1 := stk.Pop()
	println(stk.Len(), stk.Empty(), q1)
	q2 := stk.Pop()
	println(stk.Len(), stk.Empty(), q2)

}
