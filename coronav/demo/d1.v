module main

import time
import rand
// import mkuse.vpp
import cxrt.coronav

fn main() {
	coronav.init_env()
	println('aaaaa')
	coronav.post(subthr1, 0)
	coronav.post(subthr1, 0)
	coronav.post(subthr1, 0)
	for {
		time.sleep(5)
		tmstr := time.now().format_ss()
		msg := 'main thr $tmstr'
		println(msg)
	}
}

fn subthr1(arg voidptr) {
	fid := coronav.goid()
	for {
		time.sleep(2)
		tid := coronav.gettid()
		tmstr := time.now().format_ss()
		msg := 'subthr1 $fid/$tid $tmstr'
		println(msg)
	}
}

