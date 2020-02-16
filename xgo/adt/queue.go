package adt

type Queue struct {
	data []voidptr
	max  int
}

func NewQueue(max int) *Queue {
	assert(max >= 0)
	q := &Queue{}
	q.max = max
	return q
}

func (q *Queue) Push(v voidptr) bool {
	q.data = q.data.append(v)
	len := q.data.len
	if q.max > 0 && len >= q.max {
		// q.data = q.data.right(q.max) // TODO
		q.data = q.data[len-q.max:]
	}
	return true
}

func (q *Queue) Pop() voidptr {
	if q.data.len > 0 {
		v := q.data[0]
		q.data = q.data[1:]
		return v
	}
	return nil
}

func (q *Queue) Empty() bool {
	return q.data.len == 0
}
func (q *Queue) Len() int {
	return q.data.len
}
