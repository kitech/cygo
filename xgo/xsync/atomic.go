package xsync

/*
#include <stdint.h>
#include <stdbool.h>
#include <stdatomic.h>
*/
import "C"

func Addint(v *int, delta int) int {
	// return C.atomic_addint(v, delta)
	// return C.atomic_fetch_add(v, delta)
	return 0
}
func Addu64(v *u32, delta u32) {

}
func Addusize(v *usize, delta usize) {

}

func Casint(v *int, old int, new int) bool {
	return false
}
func Casu64(v *int, old int, new int) bool {
	return false
}
func Casusize(v *usize, old usize, new usize) bool {
	return false
}
func Casvptr(v *voidptr, old voidptr, new voidptr) bool {
	return false
}

func Swapint(v *int, new int) {

}

func Swapu64(v *u64, new u64) {

}
func Swapusize(v *usize, new usize) {

}
func Swapvptr(v *voidptr, new voidptr) {

}
