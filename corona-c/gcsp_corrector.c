// TODO: convert this to v code? handle all platforms
// currently if it's not macos or win it defaults to the linux impl

static void crn_sp_corrector(void** sp_ptr, void* tid) {
    size_t stack_size;
    char* stack_addr;
#ifdef _WIN64
    ULONG_PTR stack_low, stack_high;
    GetCurrentThreadStackLimits(&stack_low, &stack_high);
    stack_size = stack_high - stack_low;
    stack_addr = (char*)stack_low;
#else
    /* Darwin/Linux/Android/FreeBSD/NetBSD/OpenBSD: corona_util.h knows how to
     * ask each libc for a thread's stack (base + size). */
    int rc = thread_getstack((pthread_t)tid, (void**)&stack_addr, &stack_size);
    assert(rc == 0);
#endif
    char *sp = (char*)*sp_ptr;
    if(sp <= stack_addr || sp >= stack_addr+stack_size) {
        *sp_ptr = (void*)stack_addr;
    }
}
