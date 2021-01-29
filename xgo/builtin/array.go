package builtin

/*
typedef struct builtin__cxarray3 builtin__cxarray3;
extern builtin__cxarray3* cxarray3_new(int, int);
extern builtin__cxarray3* cxarray3_append(builtin__cxarray3* this, voidptr v);
*/
import "C"

// 定义为array3，比 array2多了类型信息
type cxarray3 struct {
	ptr    voidptr
	len    int
	cap    int
	elemsz int
	typ    *Metatype
}

//go:nodefer
//export cxarray3_new
func cxarray3_new(cap int, elemsz int) *cxarray3 {
	arr := &cxarray3{}

	arr.elemsz = elemsz
	arr.len = cap
	cap = ifelse(cap < 8, 9, cap)
	arr.cap = cap

	sz := cap * elemsz
	arr.ptr = malloc3(sz)
	return arr
}

//export cxarray3_new2
func cxarray3_new2(len int, cap int, elemsz int) *cxarray3 {
	arr := cxarray3_new(cap, elemsz)
	arr.len = len
	return arr
}

//export cxarray3_new3
func cxarray3_new3(cap int, ty *Metatype) *cxarray3 {
	arr := cxarray3_new(cap, ty.Size)
	arr.typ = ty
	return arr
}

func (arr *cxarray3) clone() *cxarray3 {
	arr2 := &cxarray3{}
	memcpy3(arr2, arr, sizeof(cxarray3))
	arr2.ptr = malloc3(arr.cap * arr.elemsz)
	if arr.len > 0 {
		memcpy3(arr2.ptr, arr.ptr, arr.len*arr.elemsz)
	}
	return arr2
}

func (arr *cxarray3) dummy() {

}

func (arr *cxarray3) each(fn func(idx int, elem voidptr)) {

}

func (arr *cxarray3) mapfn(fn func(idx int, elem voidptr) *cxarray3) {

}

func (arr *cxarray3) reduce(fn func(idx int, elem voidptr) bool) {

}
func (arr *cxarray3) filter(fn func(idx int, elem voidptr) bool) {

}

func (arr *cxarray3) Ptr() voidptr { return arr.ptr }
func (arr *cxarray3) Len() int     { return arr.len }
func (arr *cxarray3) Cap() int     { return arr.cap }
func (arr *cxarray3) Elemsz() int  { return arr.elemsz }

//export cxarray3_size
func (arr *cxarray3) size() int { return arr.len }

func (a0 *cxarray3) expand(n int) {
	assert(n > 0)
	sz := a0.len + n
	if sz > a0.cap {
		cap := a0.cap * 2
		cap = ifelse(cap < sz, sz, cap)
		ptr := realloc3(a0.ptr, cap*a0.elemsz)
		a0.ptr = ptr
		a0.cap = cap
	}
	assert(a0.cap >= a0.len+n)
}

// v need to a pointer of original var

//export cxarray3_append
func (a0 *cxarray3) append(v voidptr) *cxarray3 {
	assert(a0 != nil)
	assert(v != nil)
	a0.expand(1)
	offset := a0.len * a0.elemsz
	dstptr := voidptr(usize(a0.ptr) + usize(offset))
	memcpy3(dstptr, v, a0.elemsz)
	a0.len += 1
	return a0
}

// v only one elem

//export cxarray3_appendn
func (a0 *cxarray3) appendn(v voidptr, n int) *cxarray3 {
	assert(a0 != nil)
	assert(n > 0)
	a0.expand(n)
	//memcpy3(dstptr, v, n*a0.elemsz)
	for i := 0; i < n; i++ {
		offset := (a0.len + i) * a0.elemsz
		dstptr := voidptr(usize(a0.ptr) + usize(offset))
		memcpy3(dstptr, v, a0.elemsz)
	}
	a0.len += n
	return a0
}

func (arr *cxarray3) prepend(v voidptr) *cxarray3 {
	return arr
}

func (arr *cxarray3) insert(i int, v voidptr) *cxarray3 {
	assert(arr != nil)
	assert(v != nil)
	assert(i >= 0)
	assert(i < arr.len)
	arr.expand(1)

	C.memmove(arr.get(i+1), arr.get(i), (arr.len - i))
	arr.set(v, i, nil)

	return arr
}

//export cxarray3_clear
func (arr *cxarray3) clear() *cxarray3 {
	if arr.len == 0 {
		return arr
	}
	totsz := arr.len * arr.elemsz
	memset3(arr.ptr, 0, totsz)
	arr.len = 0
	return arr
}

//export cxarray2_clear
func (arr *cxarray3) clear2() *cxarray3 {
	if arr.len == 0 {
		return arr
	}
	totsz := arr.len * arr.elemsz
	memset3(arr.ptr, 0, totsz)
	arr.len = 0
	return arr
}

func (arr *cxarray3) join(sep string) string {
	return ""
}

//export cxarray3_slice
func (arr *cxarray3) slice(start int, end int) *cxarray3 {
	assert(arr != nil)
	assert(start >= 0)
	assert(end >= 0)
	assert(end >= start)

	newarr := cxarray3_new(end-start+1, arr.elemsz)
	newarr.typ = arr.typ
	memcpy3(newarr.ptr, voidptr(usize(arr.ptr)+usize(start)), end-start)
	newarr.len = end - start
	return newarr
}

// It takes a list as argument, and returns its first element.
func (arr *cxarray3) car() voidptr {
	return arr.get(0)
}

// It takes a list as argument, and returns a list without the first element
func (arr *cxarray3) cdr() *cxarray3 {
	if arr.len > 0 {
		return arr.slice(1, arr.len)
	}
	return nil
}

// cdr -> car
func (arr *cxarray3) cadr() voidptr {
	return nil
}

func (arr *cxarray3) first() voidptr {
	return arr.get(0)
}

// support idx < 0, then from last
//export cxarray3_get_at
func (a0 *cxarray3) get(idx int) *voidptr {
	assert(a0 != nil)
	pos := ifelse(idx < 0, a0.len+idx, idx)
	assert(pos >= 0)
	assert(pos < a0.len)

	offset := pos * a0.elemsz
	var out *voidptr
	out = (*voidptr)(usize(a0.ptr) + usize(offset))
	return out
}

// v need to a pointer of original var

//export cxarray3_replace_at
func (a0 *cxarray3) set(v voidptr, idx int, out *voidptr) voidptr {
	assert(a0 != nil)
	assert(idx >= 0)
	assert(idx < a0.len)

	offset := idx * a0.elemsz
	if out != nil {
		if v == nil {
			*out = nil
		} else {
			memcpy3(out, voidptr(usize(a0.ptr)+usize(offset)), a0.elemsz)
		}
	}
	if v == nil {
		// memcpy3(voidptr(usize(a0.ptr)+usize(offset)), &v, a0.elemsz)
		C.memset(voidptr(usize(a0.ptr)+usize(offset)), 0, a0.elemsz)
	} else {
		memcpy3(voidptr(usize(a0.ptr)+usize(offset)), v, a0.elemsz)
	}
	return out
}

// array.ptr()

//export cxarray3_delete
func (arr *cxarray3) delete(idx int) *cxarray3 {
	assert(arr != nil)
	assert(idx >= 0)
	if idx > arr.len-1 {
	} else if idx == arr.len-1 {
		arr.len -= 1
	} else {
		cpsz := (arr.len - 1 - idx) * arr.elemsz
		// offset1 := arr.ptr + idx*arr.elemsz // TODO compiler
		offset1 := voidptr(usize(arr.ptr) + usize(idx*arr.elemsz))
		offset2 := voidptr(usize(arr.ptr) + usize((idx+1)*arr.elemsz))
		C.memmove(offset1, offset2, cpsz)
		arr.len -= 1
	}
	return arr
}

//export cxarray3_reverse
func (arr *cxarray3) reverse() *cxarray3 {
	mem := malloc3(arr.elemsz)
	alen := arr.len
	for i := 0; i < alen/2; i++ {
		mi := alen - 1 - i
		offset1 := voidptr(usize(arr.ptr) + usize(i*arr.elemsz))
		offset2 := voidptr(usize(arr.ptr) + usize(mi*arr.elemsz))
		memcpy3(mem, offset1, arr.elemsz)
		memcpy3(offset1, offset2, arr.elemsz)
		memcpy3(offset2, mem, arr.elemsz)
	}

	return arr
}

//export cxarray3_left
func (arr *cxarray3) left(count int) *cxarray3 {
	if count <= 0 {
		return arr
	}

	clrsz := (arr.len - count) * arr.elemsz
	arr.len = count
	offset1 := voidptr(usize(arr.ptr) + usize((count-1)*arr.elemsz))
	memset3(offset1, 0, clrsz)

	return arr
}

//export cxarray3_right
func (arr *cxarray3) right(count int) *cxarray3 {
	if count <= 0 {
		return nil
	}

	clrsz := (arr.len - count) * arr.elemsz
	cpsz := count * arr.elemsz
	offset1 := voidptr(usize(arr.ptr) + usize((arr.len-count)*arr.elemsz))
	offset2 := voidptr(usize(arr.ptr) + usize((count-1)*arr.elemsz))
	arr.len = count
	memmove3(arr.ptr, offset1, cpsz)
	memset3(offset2, 0, clrsz)

	return arr
}

//export cxarray3_mid
func (arr *cxarray3) mid(low int, high int) *cxarray3 {
	// TODO

	return arr
}

//export cxarray3_last
func (arr *cxarray3) last() voidptr {
	if arr.len == 0 {
		return nil
	}

	offset1 := voidptr(usize(arr.ptr) + usize((arr.len-1)*arr.elemsz))
	return offset1
}

// TODO support string?

//export cxarray3_exist
func (arr *cxarray3) exist(elem voidptr) bool {
	if arr.len == 0 {
		return false
	}

	alen := arr.len
	for i := 0; i < alen; i++ {
		offset1 := voidptr(usize(arr.ptr) + usize(i*arr.elemsz))
		rv := memcmp3(offset1, elem, arr.elemsz)
		if rv == 0 {
			return true
		}
	}
	return false
}
