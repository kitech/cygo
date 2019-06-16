#include <stdio.h>
#include <time.h>
#include <stdlib.h>
#include <dlfcn.h>

#include "coronapriv.h"

void
__attribute__ ((constructor))
__attribute__((no_instrument_function))
trace_begin (void)
{
}

void
__attribute__ ((destructor))
__attribute__((no_instrument_function))
trace_end (void)
{
}



void
__attribute__((no_instrument_function))
__cyg_profile_func_enter (void *func,  void *caller)
{
    char base = 0;

    Dl_info di;
    int rv = dladdr(func, &di);
    Dl_info di2;
    int rv2 = dladdr(caller, &di2);

    printf("%s:%d func=%p caller=%p %s <- %s\n",
           __FILE__, __LINE__, func, caller, di.dli_sname, di2.dli_sname);

    fiber* gr = crn_fiber_getcur();
    if (gr == 0) return;

    void* basesp = gr!=0?gr->stack.sptr:0;
    ssize_t stksz = gr!=0 ? ((uintptr_t)&base - (uintptr_t)gr->stack.sptr) : 0;
    printf("%s:%d func=%p caller=%p %s id=%d stksz=%ld cursp=%p basesp=%p\n",
           __FILE__, __LINE__, func, caller, di.dli_sname, gr!=0?gr->id:0, stksz, &base, basesp);
    if (gr != 0) {
        assert(1==2);
    }
}

void
__attribute__((no_instrument_function))
__cyg_profile_func_exit (void *func, void *caller)
{
    char base = 0;
    // printf("%s:%d func=%p caller=%p %d\n", __FILE__, __LINE__, func, caller, 0);
}
