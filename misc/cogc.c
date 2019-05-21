#include <stdio.h>
#include <stdlib.h>
#include <stdint.h>
#include <assert.h>
#include <pthread.h>
#include <unistd.h>
#include <ucontext.h>
#include <gc.h>

#define STACK_SIZE 1024*64

ucontext_t child, parent;

struct thr_hndl_sb_s {
    void *gc_thread_handle;
    struct GC_stack_base sb;
    void *bottom;
};
struct thr_hndl_sb_s sb0 = {0};
struct thr_hndl_sb_s sb1 = {0};
struct thr_hndl_sb_s sb2 = {0};

void threadFunction() {
    GC_set_stackbottom123((void*)pthread_self(), sb1.bottom); // this works
    GC_gcollect();
    GC_MALLOC(700);

    printf("Child: Switch to parent\n");
    swapcontext( &child, &parent );
    GC_set_stackbottom123((void*)pthread_self(), sb1.bottom); // this works
    GC_gcollect();
    printf("Child: Switch to parent\n");
    swapcontext( &child, &parent );
}

extern char* GC_stackbottom;
int main() {
    GC_init();
    sb0.gc_thread_handle = GC_get_my_stackbottom(&sb0.sb);
    sb0.bottom = GC_stackbottom;
    assert(GC_stackbottom == sb0.sb.mem_base);

    void *mem = GC_MALLOC(1000);


    getcontext( &child );
    child.uc_link = 0;
    child.uc_stack.ss_sp = GC_malloc_uncollectable( STACK_SIZE );
    child.uc_stack.ss_size = STACK_SIZE;
    child.uc_stack.ss_flags = 0;
    if ( child.uc_stack.ss_sp == 0 ) {
        perror( "malloc: Could not allocate stack" );
        exit( 1 );
    }
    sb1.sb.mem_base = child.uc_stack.ss_sp;
    sb1.bottom = (void*)((uintptr_t)(sb1.sb.mem_base) + STACK_SIZE);
    makecontext( &child, &threadFunction, 0);

    printf("Parent: Switch to child\n");
    swapcontext( &parent, &child );
    GC_set_stackbottom123((void*)pthread_self(), sb0.sb.mem_base);
    printf("Parent: Switch to child\n");
    swapcontext( &parent, &child );
    GC_set_stackbottom123((void*)pthread_self(), sb0.sb.mem_base);

    GC_free( child.uc_stack.ss_sp );
    GC_gcollect();
    sleep(3);
    return 0;
}


