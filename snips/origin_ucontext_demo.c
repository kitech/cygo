#define _XOPEN_SOURCE

#include <stdio.h>
       #include <stdlib.h>
       #include <ucontext.h>

       #include <pthread.h>
       #define GC_THREADS
       #include <gc.h>


       static ucontext_t *uctx_main, *uctx_func1, *uctx_func2;

       #define handle_error(msg) \
           do { perror(msg); exit(EXIT_FAILURE); } while (0)

       static void
       func1(void)
       {
           printf("%s: started %d\n", __func__, sizeof(ucontext_t));
           printf("%s: swapcontext(&uctx_func1, &uctx_func2)\n", __func__);
           if (swapcontext(uctx_func1, uctx_func2) == -1)
               handle_error("swapcontext");
           printf("%s: returning\n", __func__);
       }

       static void
       func2(void)
       {
           printf("%s: started\n", __func__);
           printf("%s: swapcontext(&uctx_func2, &uctx_func1)\n", __func__);
           if (swapcontext(uctx_func2, uctx_func1) == -1)
               handle_error("swapcontext");
           printf("%s: returning\n", __func__);
       }


		void* thproc(void*) {
	        char* func1_stack = malloc(16384);//[16384];
           char* func2_stack = malloc(16384);// [16384];


			           if (getcontext(uctx_func1) == -1)
               handle_error("getcontext");
           uctx_func1->uc_stack.ss_sp = func1_stack;
           uctx_func1->uc_stack.ss_size = 16384;//sizeof(func1_stack);
           uctx_func1->uc_link = uctx_main;
           makecontext(uctx_func1, func1, 0);

			int argc = 1;

           if (getcontext(uctx_func2) == -1)
               handle_error("getcontext");
           uctx_func2->uc_stack.ss_sp = func2_stack;
           uctx_func2->uc_stack.ss_size = 16384 ;//sizeof(func2_stack);
           /* Successor context is f1(), unless argc > 1 */
           uctx_func2->uc_link = (argc > 1) ? NULL : uctx_func1;
           makecontext(uctx_func2, func2, 0);

           printf("%s: swapcontext(&uctx_main, &uctx_func2)\n", __func__);
           if (swapcontext(uctx_main, uctx_func2) == -1)
               handle_error("swapcontext");

           printf("%s: exiting\n", __func__);
           exit(EXIT_SUCCESS);
}

#include <unistd.h>

     int
       main(int argc, char *argv[])
       {
		   GC_INIT();
		   uctx_main = GC_MALLOC(sizeof(ucontext_t));
		   uctx_func1 = GC_MALLOC(sizeof(ucontext_t));
		   uctx_func2 = GC_MALLOC(sizeof(ucontext_t));

			pthread_t t ;
			pthread_create(&t, 0, thproc, 0);
			sleep(5);
       }
