package builtin

type mirarray struct {
	ptr      voidptr
	len      int
	cap      int
	elemsize int
}

// array.ptr()

//export cxarray2_ptr
func cxarray2_ptr(arrx voidptr) voidptr {
	var arr *mirarray
	arr = (*mirarray)(arrx)
	return arr.ptr
}
