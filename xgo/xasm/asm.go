package xasm

/*
void* cygo_asm_demo_full() {
    int a = 10;
    void* b = 0;
    __asm__ ("movl %1, %%eax; movq %%rsp, %0;"
  		:"=r"(b)		// output
  		:"r"(a)			// input
  		:"%eax"			// clobbered register
  	);
}
// but it only get SP of current state, not out scope state
void* cygo_asm_getsp() {
    void* retval = 0;
    __asm__ ("movq %%rsp, %0;"
  		:"=r"(retval)		// output
  	);
    return retval;
}
*/
import "C"

const (
	MOVL = iota + 1
	ADDL
	SUBL

	IMULL
	XADDL

	MOV
	SUB
	ADD

	LOCK

	EAX
	EBX
	ECX
	EDX
)

// AT&T 语法，指令 源, 目标
// INTERL语法，指令 目标, 源
// AT&T 寄存器带前缀 %，而 INTEL 语法不带前缀，用来区分是哪种ASM语法
// AT&T 立即数带前缀 $，而 INTEL 语法不带前缀，用来区分是哪种ASM语法

type VM interface {
	mov(src int, dst int)
	push(src int, dst int)
	pop()
}

func outasm(code string, outs ...interface{}, ins ...interface{}, clobs ...string) {
	// => __asm__(code :"=r"(out0) : "r"(in0) : "clob0")
}

func keep() {}
