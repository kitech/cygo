#include <stdio.h>
#include <stdlib.h>
#include <ucontext.h>
#include <gc.h>

#define STACK_SIZE 1024*64

ucontext_t child, parent;

void threadFunction() {
    GC_gcollect();

    printf("Child: Switch to parent\n");
    swapcontext( &child, &parent );
    printf("Child: Switch to parent\n");
    swapcontext( &child, &parent );
}

int main() {
    GC_init();
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
    makecontext( &child, &threadFunction, 0);

    printf("Parent: Switch to child\n");
    swapcontext( &parent, &child );
    printf("Parent: Switch to child\n");
    swapcontext( &parent, &child );

    GC_free( child.uc_stack.ss_sp );
    return 0;
}


