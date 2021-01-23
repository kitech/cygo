package sched

/*
#include <stdio.h>
#include <pthread.h>

__thread int co_sched_mcid = -1;
__thread int co_sched_grid = -1;
__thread void* co_sched_mcobj = 0;
__thread void* co_sched_grobj = 0;


void temp_print_sched(int which) {
switch (which) {
case 0:
printf("sched pregc init ...\n");
break;
case 1:
printf("sched pregc init done\n");
break;
default:
printf("sched wtt %d\n", which);
break;
}

}
*/
import "C"

import (
	"iohook"
	"rtcom"
)

func Keepme() {
	if false {
		rtcom.Keepme()
		iohook.Keepme()
	}
}

// gc not inited
func pre_gc_init() {
	// cannot alloc here
	C.temp_print_sched(0)
	// C.printf("sched pregc init\n") // TODO
	//yielder := rtcom.Yielder{} // TODO
	// yielder := &rtcom.Yielder{}
	// yielder.incoro = incoro
	// yielder.getcoro = getcoro
	// yielder.yield = onyield
	// yielder.yield_multi = onyield_multi
	// resumer := &rtcom.Resumer{}
	// resumer.resume_one = onresume
	// rtcom.pre_gc_init(&yielder, &resumer, (voidptr)(0))
	// iohook.pre_main_init(rtcom.yielder(), (voidptr)(0))

	rtcom.pre_gc_init(incoro, getcoro, onyield, onyield_multi,
		onresume, nil, nil, nil, nil)
	C.temp_print_sched(1)
	iohook.pre_main_init(incoro, getcoro, onyield, onyield_multi,
		nil, nil, nil, nil)
	C.temp_print_sched(1)
}

// gc inited
func pre_main_init() {
	println("sched premain init")
	yielder := rtcom.yielder()
	resumer := rtcom.resumer()
	// iopoller.pre_main_init(resumer)
	// chan1.pre_main_init(yielder, resumer, voidptr(0))
	println("y&r", yielder, resumer)
}
func post_main_deinit() {
	// TODO需要退出coroutine以及procer
}

//////////////////

func incoro() int {
	mcid := getmcid()
	return mcid
}
func getcoro() voidptr {
	return C.co_sched_grobj
}

func onyield(fdns i64, ytype int) int {

	return 0
}

func onyield_multi(ytype int, cnt int, fds *i64, ytypes *int) int {
	return 0
}

func onresume(grx voidptr, ytype int, grid int, mcid int) {

}

/////////////////////

func getmcid() int { return C.co_sched_mcid }
