package adt

func test_q1() {
	q := NewQueue(3)
	println(q.Len(), q.Empty())
	q.Push(q)
	q.Push(q)
	q.Push(q)
	q.Push(q)
	q.Push(q)
	println(q.Len(), q.Empty())

}
