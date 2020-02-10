package builtin

/*
// #define builtin_sizeof(x) sizeof(x)
*/
import "C"

func htkey_cmp_int(k1 voidptr, k2 voidptr) bool {
	var k1p *int = (*int)(k1)
	var k2p *int = (*int)(k2)
	return *k1p == *k2p
}

func htkey_hash_int32(k1 voidptr, len int) usize {
	var k1p *uint32 = (*int32)(k1)
	var x = *k1p

	x = ((x >> 16) ^ x) * 0x45d9f3b
	x = ((x >> 16) ^ x) * 0x45d9f3b
	x = (x >> 16) ^ x

	return x
}
func htkey_unhash_int32(hv usize) int32 {
	var x uint32 = hv
	x = ((x >> 16) ^ x) * 0x119de1f3
	x = ((x >> 16) ^ x) * 0x119de1f3
	x = (x >> 16) ^ x
	return x
}
func htkey_hash_int64(k1 voidptr, len int) usize {
	var k1p *uint64 = (*int64)(k1)
	var x = *k1p

	x = (x ^ (x >> 30)) * (0xbf58476d1ce4e5b9)
	x = (x ^ (x >> 27)) * (0x94d049bb133111eb)
	x = x ^ (x >> 31)

	return x
}
func htkey_unhash_int64(hv usize) int64 {
	var x uint64 = hv
	x = (x ^ (x >> 31) ^ (x >> 62)) * (0x319642b2d24d8ec3)
	x = (x ^ (x >> 27) ^ (x >> 54)) * (0x96de1b173f119089)
	x = x ^ (x >> 30) ^ (x >> 60)
	return x
}

// len = 0
func htkey_hash_int(k1 voidptr, len int) usize {
	var k1p *int = (*int)(k1)
	var k1v = *k1p

	tysz := sizeof(int(0))
	if tysz == 4 {
		return htkey_hash_int32(k1, tysz)
	} else {
		return htkey_hash_int64(k1, tysz)
	}

	return 0
}

/*******************************************************************************
 *
 *
 *  djb2 string hash
 *
 *
 ******************************************************************************/

// k1 should be cstring of byteptr
func htkey_hash_str(k1 voidptr, len int) usize {
	var k1p byteptr = (byteptr)(k1)
	var hash usize
	hash = 0 + 5381 + len + 1
	for i := 0; i < len; i++ {
		c := k1p[i]
		hash = ((hash << 5) + hash) ^ usize(c)
	}

	return hash
}

// len = 0
func htkey_hash_ptr(k1 voidptr, len int) usize {
	// tysz := sizeof(voidptr) // TODO compiler
	tysz := sizeof(voidptr(0))
	if tysz == 4 {
		return htkey_hash_int32(&k1, 0)
	} else {
		return htkey_hash_int64(&k1, 0)
	}
	return 0
}

func htkey_cmp_getfunc(kind int) func(k1 voidptr, k2 voidptr) bool {
	switch kind {
	case Int:
		return htkey_cmp_int
	}
	return nil
}

func htkey_hash_getfunc(kind int) func(k1 voidptr, len int) usize {
	switch kind {
	case Int:
		return htkey_hash_int
	}
	return nil
}

func typesize(kind int) int {
	switch kind {
	case Int:
		// return sizeof(int) // TODO compiler
		return sizeof(int(0))
	case Voidptr:
		return sizeof(voidptr(0))
	}
	return -1
}

type mapnode struct {
	key  voidptr
	val  voidptr
	next *mapnode
}

type mirmap struct {
	len_ int
	cap_ int
	ptr  voidptr

	keykind int
	valkind int
	keysz   int
	valsz   int
	hashfn  func(k1 voidptr) uint64
	cmpfn   func(k1 voidptr, k2 voidptr) bool

	bucketsz  int
	threshold int
}

func mirmap_new(keykind int) *mirmap {
	mp := &mirmap{}
	mp.keykind = keykind
	mp.bucketsz = 16
	mp.cap_ = 1
	var vptrsz int = sizeof(voidptr(0))
	mp.ptr = malloc3(1 * vptrsz)
	return mp
}

func (mp *mirmap) dummy() {

}

//export mirmap_dummy2
func (mp *mirmap) dummy2() {}

func (mp *mirmap) each(fn func(key voidptr, val voidptr)) {

}

func (mp *mirmap) mapfn(fn func(key voidptr, val voidptr) *mirmap) {

}

func (mp *mirmap) reduce(fn func(key voidptr, val voidptr) bool) {

}

func (mp *mirmap) len() int {
	return 0
}

func (mp *mirmap) cap() int {
	return 0
}

// [0,100]
func (mp *mirmap) useratio() int {
	r := mp.len_ * 100 / mp.cap_
	return r
}

func (mp *mirmap) haskey(k voidptr) bool {
	return false
}

func (mp *mirmap) delete(k voidptr) bool {
	return false
}

func (mp *mirmap) insert(k voidptr, v voidptr) bool {

	return false
}

func (mp *mirmap) access1(k voidptr) voidptr {
	return nil
}

func (mp *mirmap) access2(k voidptr) (voidptr, bool) {
	return nil, false
}

func (mp *mirmap) expand() {
	return
}

func (mp *mirmap) rehash() {
	return
}
