package cjson

/*
#cgo CFLAGS: -IPKGDIR/
#cgo CFLAGS: -I/home/me/oss/src/cxrt/xgo/cjson/

#include "/home/me/oss/src/cxrt/xgo/cjson/cJSON.h"

*/
import "C"
import "xgo/xerrors"

type cjson struct {
	next  *cjson
	prev  *cjson
	child *cjson

	typ int

	valstr byteptr
	valint int
	valdbl float64

	namestr byteptr
}

type Json struct {
	cobj *cjson
}

func version() string {
	rv := C.cJSON_Version()
	return gostring(rv)
}

func parse(s string) (*Json, error) {
	cobj := C.cJSON_Parse(s.ptr)
	var err error
	if cobj == nil {
		eraw := C.cJSON_GetErrorPtr()
		elen := strlen3(eraw)
		epos := s.len - elen - 1
		estr := "parse error at " + epos.repr() + " `" + string(s.ptr[epos]) + "`"
		err = xerrors.New(estr)
	}
	nilreturn(cobj, nil, err)

	jso := &Json{}
	jso.cobj = cobj
	set_finalizer(jso, cjson_dtor)
	return jso, nil
}
func cjson_dtor(ptr voidptr) {
	var jo *Json = ptr
	C.cJSON_Delete(jo.cobj)
}

func (jo *Json) kind() int {
	return jo.cobj.typ
}

func (jo *Json) at(i int) *Json {
	if jo.cobj.typ != C.cJSON_Array {
		println("not support at()")
		return nil
	}

	item := C.cJSON_GetArrayItem(jo.cobj, i)
	nilreturn(item, nil)
	jn := &Json{item}
	return jn
}

func (jo *Json) size() int {
	switch jo.cobj.typ {
	case C.cJSON_Array, C.cJSON_Object:
		rv := C.cJSON_GetArraySize(jo.cobj)
		return rv
	}
	return -1
}

func (jo *Json) by(key string) *Json {
	if jo.cobj.typ != C.cJSON_Object {
		return nil
	}
	item := C.cJSON_GetObjectItem(jo.cobj, key.ptr)
	nilreturn(item, nil)
	jn := &Json{item}
	return jn
}
func (jo *Json) exist(key string) bool {
	if jo.cobj.typ != C.cJSON_Object {
		return false
	}
	ok := C.cJSON_HasObjectItem(jo.cobj, key.ptr)
	return ok
}

// only current node
func (jo *Json) repr() string {
	nilreturn(jo, "<jsnil>")
	if true {
		rv := C.cJSON_Print(jo.cobj)
		s := gostring_clone(rv)
		C.cJSON_free(rv)
		return s
	}

	switch jo.cobj.typ {
	case C.cJSON_True:
		return "jstrue"
	case C.cJSON_False:
		return "jsfalse"
	case C.cJSON_NULL:
		return "jsnull"
	case C.cJSON_Number:
		v := jo.cobj.valdbl
		return v.repr()
	case C.cJSON_String:
		v := jo.cobj.valstr
		return gostring(v)
	case C.cJSON_Array:
		return "jsarray"
	case C.cJSON_Object:
		return "jsobject"
	}
	return "jsunk"
}

// keys only
func (jo *Json) pathby(keys ...string) *Json {
	return nil
}

// keys and indexes
func (jo *Json) pathat(kois ...interface{}) *Json {
	return nil
}

func (jo *Json) bebool() bool {
	cobj := jo.cobj
	if cobj.typ == C.cJSON_True {
		return true
	}
	return false
}

func (jo *Json) beint() int {
	cobj := jo.cobj
	if cobj.typ == C.cJSON_Number {
		return cobj.valint
	}
	return 0
}

func (jo *Json) bef64() float64 {
	cobj := jo.cobj
	if cobj.typ == C.cJSON_Number {
		return cobj.valdbl
	}
	return 0
}

func (jo *Json) bestr() string {
	cobj := jo.cobj
	if cobj.typ == C.cJSON_String {
		return gostring(cobj.valstr)
	}
	return ""
}

// array/object
func (jo *Json) eachao(fn func(kind int, index int, key string, val *Json)) {
	cobj := jo.cobj
	jot := &Json{}
	var ekey string

	var idx = -1
	var item *cjson
	// item = ifelse(cobj != nil, cobj.child, nil) // TODO compiler
	item = ifelse(cobj != nil, cobj.child, item)
	for item = cobj.child; item != nil; item = item.next {
		idx++
		jot.cobj = item
		println(item.typ, jot.repr(), item.namestr)
		if fn == nil {
			continue
		}
		if item.namestr != nil {
			fn(cobj.typ, idx, gostring(item.namestr), jot)
		} else {
			fn(cobj.typ, idx, ekey, jot)
		}
	}

}

func (jo *Json) eacharr(fn func(idx int, val *Json)) {
	cobj := jo.cobj
	jot := &Json{}

	var item *cjson
	// item = ifelse(cobj != nil, cobj.child, nil) // TODO compiler
	item = ifelse(cobj != nil, cobj.child, item)
	idx := -1
	for item = cobj.child; item != nil; item = item.next {
		idx++
		jot.cobj = item
		fn(idx, jot)
	}
}

func (jo *Json) eachobj(fn func(key string, val *Json)) {
	cobj := jo.cobj
	jot := &Json{}

	var item *cjson
	// item = ifelse(cobj != nil, cobj.child, nil) // TODO compiler
	item = ifelse(cobj != nil, cobj.child, item)
	for item = cobj.child; item != nil; item = item.next {
		jot.cobj = item
		key := gostring(item.namestr)
		fn(key, jot)
	}
}

func Keep() {}
