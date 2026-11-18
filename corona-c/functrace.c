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
    const char* sname1 = di.dli_sname != 0 ? di.dli_sname : "?";
    const char* sname2 = di2.dli_sname != 0 ? di2.dli_sname : "?";

    printf("%s:%d func=%p caller=%p %s <- %s\n",
           __FILE__, __LINE__, func, caller, sname1, sname2);

    fiber* gr = crn_fiber_getcur();
    if (gr == 0) return;

    void* basesp = gr!=0?gr->stack.sptr:0;
    ssize_t stksz = gr!=0 ? ((uintptr_t)&base - (uintptr_t)gr->stack.sptr) : 0;
    printf("%s:%d func=%p caller=%p %s id=%d stksz=%ld cursp=%p basesp=%p\n",
           __FILE__, __LINE__, func, caller, sname1, gr!=0?gr->id:0, stksz, &base, basesp);
    if (gr != 0) {
        // keep the original abort semantics, but log before dying so the
        // trace (fiber id, stack usage) survives in the log
        lwarn("instraced func in fiber %d stksz=%ld cursp=%p basesp=%p\n",
              gr->id, stksz, &base, basesp);
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
