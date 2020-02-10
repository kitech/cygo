package builtin

/*
 */
import "C"

type mirarray struct {
	ptr    voidptr
	len    int
	cap    int
	elemsz int
}

func mirarray_new() *mirarray {
	arr := &mirarray{}
	return arr
}

func (arr *mirarray) dummy() {

}

func (arr *mirarray) each(fn func(idx int, elem voidptr)) {

}

func (arr *mirarray) mapfn(fn func(idx int, elem voidptr) *mirarray) {

}

func (arr *mirarray) reduce(fn func(idx int, elem voidptr) bool) {

}
func (arr *mirarray) filter(fn func(idx int, elem voidptr) bool) {

}

func (arr *mirarray) Ptr() voidptr {
	return arr.ptr
}
func (arr *mirarray) Len() int {
	return arr.len
}
func (arr *mirarray) Cap() int {
	return arr.cap
}
func (arr *mirarray) Elemsz() int {
	return arr.elemsz
}

func (arr *mirarray) delete(idx int) *mirarray {
	return arr
}

func (arr *mirarray) append() *mirarray {
	return arr
}

func (arr *mirarray) reverse() *mirarray {
	return arr
}

func (arr *mirarray) clear() *mirarray {
	return arr
}

func (arr *mirarray) join(sep string) string {
	return ""
}

// array.ptr()

//export cxarray2_ptr
func cxarray2_ptr(arrx voidptr) voidptr {
	var arr *mirarray
	arr = (*mirarray)(arrx)
	return arr.ptr
}

//export cxarray2_delete
func cxarray2_delete(arrx voidptr, idx int) voidptr {
	assert(arrx != nil)
	assert(idx >= 0)
	var arr *mirarray
	arr = (*mirarray)(arrx)
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
	return arrx
}

//export cxarray2_clear
func cxarray2_clear(arrx voidptr) voidptr {
	var arr *mirarray
	arr = (*mirarray)(arrx)

	alen := arr.len
	arr.len = 0

	opsz := alen * arr.elemsz
	C.memset(arr.ptr, 0, opsz)
	return arrx
}

//export cxarray2_reverse
func cxarray2_reverse(arrx voidptr) voidptr {
	var arr *mirarray
	arr = (*mirarray)(arrx)

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

	return arrx
}

//export cxarray2_left
func cxarray2_left(arrx voidptr, count int) voidptr {
	if count <= 0 {
		return arrx
	}

	var arr *mirarray
	arr = (*mirarray)(arrx)

	clrsz := (arr.len - count) * arr.elemsz
	arr.len = count
	offset1 := voidptr(usize(arr.ptr) + usize((count-1)*arr.elemsz))
	memset3(offset1, 0, clrsz)

	return arrx
}

//export cxarray2_right
func cxarray2_right(arrx voidptr, count int) voidptr {
	if count <= 0 {
		return arrx
	}

	var arr *mirarray
	arr = (*mirarray)(arrx)

	clrsz := (arr.len - count) * arr.elemsz
	cpsz := count * arr.elemsz
	offset1 := voidptr(usize(arr.ptr) + usize((arr.len-count)*arr.elemsz))
	offset2 := voidptr(usize(arr.ptr) + usize((count-1)*arr.elemsz))
	arr.len = count
	memmove3(arr.ptr, offset1, cpsz)
	memset3(offset2, 0, clrsz)

	return arrx
}

//export cxarray2_mid
func cxarray2_mid(arrx voidptr, low int, high int) voidptr {
	var arr *mirarray
	arr = (*mirarray)(arrx)

	// TODO

	return arrx
}

//export cxarray2_last
func cxarray2_last(arrx voidptr) voidptr {
	var arr *mirarray
	arr = (*mirarray)(arrx)

	if arr.len == 0 {
		return nil
	}

	offset1 := voidptr(usize(arr.ptr) + usize((arr.len-1)*arr.elemsz))
	return offset1
}

// TODO support string?

//export cxarray2_has
func cxarray2_has(arrx voidptr, elem voidptr) bool {
	var arr *mirarray
	arr = (*mirarray)(arrx)

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

// TODO
func cxarray2_append(arrx voidptr, elem voidptr) voidptr {
	//arrx = append(arrx, elem)
	return arrx
}
