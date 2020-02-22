package builtin

/*
// #define builtin_sizeof(x) sizeof(x)
*/
import "C"

func htkey_eq_int32(k1 voidptr, k2 voidptr) bool {
	var k1p *int32 = (*int32)(k1)
	var k2p *int32 = (*int32)(k2)
	return *k1p == *k2p
}

func htkey_eq_int64(k1 voidptr, k2 voidptr) bool {
	var k1p *int64 = (*int64)(k1)
	var k2p *int64 = (*int64)(k2)
	return *k1p == *k2p
}

func htkey_eq_f32(k1 voidptr, k2 voidptr) bool {
	var k1p *float32 = (*float32)(k1)
	var k2p *float32 = (*float32)(k2)
	return *k1p == *k2p
}
func htkey_eq_f64(k1 voidptr, k2 voidptr) bool {
	var k1p *float64 = (*float64)(k1)
	var k2p *float64 = (*float64)(k2)
	return *k1p == *k2p
}

func htkey_eq_ptr(k1 voidptr, k2 voidptr) bool {
	return k1 == k2
}

// nouse
func htkey_eq_str(k1 voidptr, k2 voidptr) bool {
	return k1 == k2
}
func htkey_eq_str2(k1 string, k2 string) bool {
	return k1 == k2
}

///
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
func htkey_hash_str2(k1 *string, len int) usize {
	k2 := *k1
	return htkey_hash_str(k2.ptr, k2.len)
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

// len = 0
func htkey_hash_f32(k1 voidptr, len int) usize {
	tysz := sizeof(float32(0))
	return htkey_hash_int32(k1, tysz)
}

// len = 0
func htkey_hash_f64(k1 voidptr, len int) usize {
	tysz := sizeof(float64(0))
	return htkey_hash_int64(k1, tysz)
}

func typesize(kind int) int {
	switch kind {
	case Int:
		// return sizeof(int) // TODO compiler
		return sizeof(int(0))
	case Voidptr:
		return sizeof(voidptr(uintptr(0)))
	}
	return -1
}

type mapnode struct {
	key  voidptr
	val  voidptr
	hash usize
	next *mapnode
}
type mapiter struct {
	mobj *mirmap
	pvec int // pos in vector
	plst int // pos in linked list
}

type mirmap struct {
	len_ int
	cap_ int
	ptr  *voidptr

	keykind int
	valkind int
	keysz   int
	valsz   int

	keyalg *typealg

	keytyp *Metatype
	valtyp *Metatype

	bucketsz  int
	threshold int
	expcnt    int
}

//export cxhashtable3_new
func mirmap_new(keykind int, valkind int) *mirmap {
	assert(keykind > 0)
	assert(valkind > 0)
	mp := &mirmap{}
	mp.keykind = keykind
	mp.valkind = valkind
	mp.bucketsz = 16
	mp.cap_ = 1
	var vptrsz int = sizeof(voidptr(0))
	mp.ptr = malloc3(1 * vptrsz)
	mp.initkeyalg(keykind)
	mp.initvalsz(valkind)
	return mp
}

func (mp *mirmap) initkeyalg(keykind int) {
	alg := &typealg{}
	switch keykind {
	case Int, Uint:
		tysz := sizeof(int(0))
		mp.keysz = tysz
		if tysz == 4 {
			alg.hash = htkey_hash_int32
			alg.equal = htkey_eq_int32
		} else {
			alg.hash = htkey_hash_int64
			alg.equal = htkey_eq_int64
		}
	case Int64, Uint64:
		alg.hash = htkey_hash_int64
		alg.equal = htkey_eq_int64
		mp.keysz = sizeof(int64(0))
	case Float32:
		alg.hash = htkey_hash_f32
		alg.equal = htkey_eq_f32
		mp.keysz = sizeof(float32(0))
	case Float64:
		alg.hash = htkey_hash_f64
		alg.equal = htkey_eq_f64
		mp.keysz = sizeof(float64(0))
	case String:
		alg.hash = htkey_hash_str2
		alg.equal = htkey_eq_str2
		mp.keysz = sizeof(voidptr(0))
	case Voidptr, Uintptr:
		alg.hash = htkey_hash_ptr
		alg.equal = htkey_eq_ptr
		mp.keysz = sizeof(voidptr(0))
	default:
		assert(1 == 2)
	}
	assert(alg.hash != nil)
	mp.keyalg = alg
}

func (mp *mirmap) initvalsz(valkind int) {
	switch valkind {
	case Int, Uint:
		mp.valsz = sizeof(int(0))
	case Int64, Uint64:
		mp.valsz = sizeof(int64(0))
	case Float32:
		mp.valsz = sizeof(float32(0))
	case Float64:
		mp.valsz = sizeof(float64(0))
	case String:
		mp.valsz = sizeof(voidptr(0))
	case Voidptr, Uintptr:
		mp.valsz = sizeof(voidptr(0))
	case Int16, Uint16:
		mp.valsz = sizeof(int16(0))
	case Bool:
		mp.valsz = sizeof(int(0))
	default:
		assert(1 == 2)
	}
	assert(mp.valsz != 0)
}

func (mp *mirmap) dummy() {

}

//export mirmap_dummy2
func (mp *mirmap) dummy2() {}

func (mp *mirmap) itnew() *mapiter {
	mit := &mapiter{}
	mit.mobj = mp
	return mit
}

func (it *mapiter) next() *mapnode {
	optr := it.mobj.ptr
	ocap := it.mobj.cap_

	for i := it.pvec; i < ocap; i++ {
		var node *mapnode = optr[i]
		for j := 0; node != nil; j++ {
			if j == it.plst {
				it.plst += 1
				return node
			}
			node = node.next
		}
		it.pvec += 1
		it.plst = 0
	}
	return nil
}

func (mp *mirmap) each(fn func(key voidptr, val voidptr)) {
	optr := mp.ptr
	ocap := mp.cap_
	for i := 0; i < ocap; i++ {
		var node *mapnode = optr[i]
		for node != nil {
			fn(node.key, node.val)
			node = node.next
		}
	}
}

// TODO fn return map's key/value type?
func (mp *mirmap) mapfn(fn func(key voidptr, val voidptr) *mirmap) *mirmap {
	res := mirmap_new(mp.keykind, mp.valkind)
	optr := mp.ptr
	ocap := mp.cap_
	for i := 0; i < ocap; i++ {
		var node *mapnode = optr[i]
		for node != nil {
			rets := fn(node.key, node.val)
			if rets != nil {
				for j := 0; j < rets.cap_; j++ {
					var n1 *mapnode = rets.ptr[j]
					for n1 != nil {
						res.insert(n1.key, n1.val)
						n1 = n1.next
					}
				}
			}

			node = node.next
		}
	}
	return res
}

func (mp *mirmap) reduce(fn func(key voidptr, val voidptr) bool) *mirmap {
	res := mirmap_new(mp.keykind, mp.valkind)

	optr := mp.ptr
	ocap := mp.cap_
	for i := 0; i < ocap; i++ {
		var node *mapnode = optr[i]
		for node != nil {
			ok := fn(node.key, node.val)
			if ok {
				res.insert(node.key, node.val)
			}
			node = node.next
		}
	}

	return res
}

func (mp *mirmap) keys() []voidptr {
	res := []voidptr{}
	optr := mp.ptr
	ocap := mp.cap_
	for i := 0; i < ocap; i++ {
		var node *mapnode = optr[i]
		for node != nil {
			res = append(res, node.key)
			node = node.next
		}
	}
	return res
}

func (mp *mirmap) keys2() []voidptr {
	res := []voidptr{}
	// TODO compiler
	/*
		mp.each(func(key voidptr, val voidptr) {
			res = append(res, key)
		})
	*/
	return res
}

func (mp *mirmap) values() []voidptr {
	res := []voidptr{}
	optr := mp.ptr
	ocap := mp.cap_
	for i := 0; i < ocap; i++ {
		var node *mapnode = optr[i]
		for node != nil {
			res = append(res, node.val)
			node = node.next
		}
	}
	return res
}
func (mp *mirmap) values2() []voidptr {
	res := []voidptr{}
	// TODO compiler
	//*
	mp.each(func(key voidptr, val voidptr) {
		res = append(res, val)
	})
	//*/
	return res
}

//export cxhashtable3_size
func (mp *mirmap) len() int {
	return mp.len_
}
func (mp *mirmap) cap() int {
	return mp.cap_
}

// [0,100]
func (mp *mirmap) useratio() int {
	r := mp.len_ * 100 / mp.cap_
	return r
}

// k need to a pointer of original var
// if direct use, mp.exist(&somekey)
func (mp *mirmap) exist(k voidptr) bool {
	return mp.haskey(k)
}

// k need to a pointer of original var
// if direct use, mp.haskey(&somekey)
func (mp *mirmap) haskey(k voidptr) bool {
	ocap := mp.cap_
	optr := mp.ptr

	fnptr := mp.keyalg.hash
	hash := fnptr(k, mp.keysz)
	idx := hash % usize(ocap)

	var onode *mapnode = optr[idx]
	for onode != nil {
		if onode.hash == hash {
			return true
		}
		onode = onode.next
	}

	return false
}

// k need to a pointer of original var

//export cxhashtable3_get
func (mp *mirmap) access0(k voidptr, v *voidptr) int {
	v1 := mp.access1(k)
	// when key not exist, v1 will nil
	if v1 != nil {
		memcpy3(v, v1, mp.valsz)
	}
	return 0
}

// k need to a pointer of original var
// return (valtype*), like, (int*)
func (mp *mirmap) access1(k voidptr) voidptr {
	ocap := mp.cap_
	optr := mp.ptr

	fnptr := mp.keyalg.hash
	hash := fnptr(k, mp.keysz)
	idx := hash % usize(ocap)

	var onode *mapnode = optr[idx]
	for onode != nil {
		if onode.hash == hash {
			return onode.val
		}
		onode = onode.next
	}

	return nil
}

//export cxhashtable3_get2 // TODO compiler
func (mp *mirmap) get2(k voidptr, out *voidptr) bool {
	res, ok := mp.access2(k)
	*out = res
	return ok
}
func (mp *mirmap) access2(k voidptr) (voidptr, bool) {
	ocap := mp.cap_
	optr := mp.ptr

	fnptr := mp.keyalg.hash
	hash := fnptr(k, mp.keysz)
	idx := hash % usize(ocap)

	var onode *mapnode = optr[idx]
	for onode != nil {
		if onode.hash == hash {
			return onode.val, true
		}
		onode = onode.next
	}

	return nil, false
}

func (mp *mirmap) clear() bool {
	mp.len_ = 0
	memset3(mp.ptr, 0, mp.cap_*sizeof(voidptr(0)))
	return true
}

//export cxhashtable3_remove
func (mp *mirmap) delete(k voidptr) bool {
	fnptr := mp.keyalg.hash
	hash := fnptr(k, mp.keysz)
	idx := hash % usize(mp.cap_)

	optr := mp.ptr
	ocap := mp.cap_
	for i := 0; i < ocap; i++ {
		var node *mapnode = optr[i]
		var prev *mapnode
		for node != nil {
			if node.hash == hash {
				if prev == nil {
					optr[i] = node.next
				} else {
					prev.next = node.next
				}
				mp.len_--
				return true
			}
			prev = node
			node = node.next
		}
	}
	return false
}

// k,v must pointer of orignal var

//export cxhashtable3_add
func (mp *mirmap) insert(k voidptr, v voidptr) bool {
	mp.expand()

	// mp.keyalg.hash(&k, mp.keysz) // TODO compiler
	fnptr := mp.keyalg.hash
	hash := fnptr(k, mp.keysz)
	idx := hash % usize(mp.cap_)
	// println(idx, hash, mp.cap_)

	var onode *mapnode = mp.ptr[idx]
	for onode != nil {
		if onode.hash == hash {
			// println("replace", idx, k)
			memcpy3(onode.val, v, mp.valsz)
			return true
		}
		onode = onode.next
	}

	node := &mapnode{}
	// node.key = k
	// node.val = v
	node.key = malloc3(mp.keysz)
	memcpy3(node.key, k, mp.keysz)
	node.val = malloc3(mp.valsz)
	memcpy3(node.val, v, mp.valsz)
	node.hash = hash
	node.next = mp.ptr[idx]

	// println("insert", mp.len_, idx, mp.cap_, node)
	mp.ptr[idx] = node
	mp.len_++

	return true
}

func (mp *mirmap) expand() {
	len := mp.len_
	if len <= mp.threshold {
		return
	}

	cap := mp.cap_ << 1
	// println("map need expand", cap>>1, "to", cap)
	ptr := malloc3(cap * sizeof(voidptr(0)))
	mp.move_nodes(ptr, cap)
	mp.ptr = ptr
	mp.cap_ = cap
	mp.threshold = 70 * cap / 100
	mp.expcnt++

	return
}

func (mp *mirmap) move_nodes(ptr *voidptr, cap int) {
	optr := mp.ptr
	ocap := mp.cap_

	for i := 0; i < ocap; i++ {
		var node *mapnode = optr[i]
		for node != nil {
			next := node.next
			idx := node.hash % usize(cap)

			node.next = ptr[idx]
			ptr[idx] = node

			node = next
		}
	}
}

func (mp *mirmap) dump() {
	optr := mp.ptr
	ocap := mp.cap_
	println("mapdmp", "len", mp.len_, "cap", ocap)
	cnter := 0
	for i := 0; i < ocap; i++ {
		var node *mapnode = optr[i]
		for j := 0; node != nil; j++ {
			cnter++
			println(i, j, cnter, node.hash)
			node = node.next
		}
		if cnter >= mp.len_ {
			break
		}
	}
	return
}

func (mp *mirmap) dumpmin() {
	ocap := mp.cap_
	println("mapdmp", "len", mp.len_, "cap", ocap, "expcnt", mp.expcnt)
}

func (mp *mirmap) chklinked() {
	optr := mp.ptr
	ocap := mp.cap_
	println("mapdmp", "len", mp.len_, "cap", ocap)
	cnter := 0
	totlnk := 0
	for i := 0; i < ocap; i++ {
		var node *mapnode = optr[i]
		var linked = 0
		for j := 0; node != nil; j++ {
			cnter++
			// println(i, j, cnter, node.hash)
			node = node.next
			if j > 0 {
				linked++
				totlnk++
			}
		}
		if linked > 0 {
			println(i, cnter, totlnk, linked)
		}
		if cnter >= mp.len_ {
			break
		}
	}
	return
}
