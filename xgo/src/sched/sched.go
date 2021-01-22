package sched

/*
#include <stdio.h>

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

//////////////////

func incoro() int {
	return 0
}

func getcoro() voidptr {
	return nil
}

func onyield() {

}

func onyield_multi() {

}

func onresume() {

}

/////////////////////
