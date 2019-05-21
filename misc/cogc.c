#define GC_THREADS

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
extern char* GC_stackbottom;
// extern void GC_register_my_thread(void*); // -DGC_THREADS

// 恢复到线程原来的栈
void* setbottom0(void*arg) {
    GC_set_stackbottom(sb2.gc_thread_handle, &sb2.sb);
    // GC_stackbottom = sb2.bottom;
    return 0;
}
// coroutine动态栈
void* setbottom1(void*arg) {
    sb1.sb.mem_base = sb1.bottom;
    GC_set_stackbottom(sb2.gc_thread_handle, &sb1.sb);
    // GC_stackbottom = sb1.bottom;
    return 0;
}

void threadFunction() {
    GC_call_with_alloc_lock(setbottom1, 0);

    GC_gcollect();
    GC_MALLOC(700);
    GC_MALLOC(3567);
    GC_MALLOC(21230);
    GC_gcollect(); // 这个地方能回收，但并不想在这个地方调用。而且为啥在中间加了sleep，回收数就变了

    printf("Child: Switch to parent\n");
    GC_call_with_alloc_lock(setbottom0, 0);
    swapcontext( &child, &parent );

    for (int i = 0; i < 900; i++) {
        GC_call_with_alloc_lock(setbottom1, 0);
        GC_MALLOC(6700);
        sleep(1);
        GC_MALLOC(3567);
        // GC_gcollect();

        printf("Child: Switch to parent2, %d\n", i);
        GC_call_with_alloc_lock(setbottom0, 0);
        swapcontext( &child, &parent );
    }
}


void* main2th(void* arg) {
    GC_get_stack_base(&sb2.sb);
    GC_register_my_thread(&sb2.sb);

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
    // 连接在当前栈与子栈之前跳转多次
    for (int i = 0; i < 900; i ++) {
        printf("Parent: Switch to child2 %d\n", i);
        swapcontext( &parent, &child );
    }

    GC_free( child.uc_stack.ss_sp );
    GC_gcollect();
    printf("thread done???\n");
}

pthread_t mainth;
int main() {
    GC_init();
    GC_allow_register_threads();

    sb0.gc_thread_handle = GC_get_my_stackbottom(&sb0.sb);
    sb0.bottom = GC_stackbottom;
    assert(GC_stackbottom == sb0.sb.mem_base);

    pthread_create(&mainth, 0, main2th, 0);
    pthread_join(mainth, 0);
    sleep(30);
    return 0;
}


