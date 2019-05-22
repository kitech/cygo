{.compile:"coro.c"}
{.compile:"corowp.c"}
{.passc:"-D_GNU_SOURCE".}
{.passc:"-DCORO_UCONTEXT".}
{.passc:"-DHAVE_UCONTEXT_H".}
{.passc:"-DHAVE_SETJMP_H".}
{.passc:"-DHAVE_SIGALTSTACK".}

type coro_context = pointer

proc coro_context_new():coro_context {.importc:"corowp_context_new".}
proc coro_create(ctx:coro_context, corofn : pointer, arg:pointer, sptr: pointer, ssze:uint) {.importc:"corowp_create".}
proc coro_create(ctx:coro_context, corofn : proc(a:pointer), arg:pointer, sptr: pointer, ssze:uint) {.importc:"corowp_create".}
proc coro_transfer(prev, next : coro_context) {.importc:"corowp_transfer".}
proc coro_destroy(ctx : coro_context) {.importc:"corowp_destroy".}

# stack
type
    coro_stack = ptr object
        sptr*: pointer
        ssze*: uint
        valgrind_id*: int

proc coro_stack_new() : coro_stack {.importc:"corowp_stack_new".}
proc coro_stack_alloc(stk:coro_stack, size: uint) : coro_stack {.importc:"corowp_stack_alloc".}
proc coro_stack_free(stk:coro_stack)  {.importc:"corowp_stack_free".}

