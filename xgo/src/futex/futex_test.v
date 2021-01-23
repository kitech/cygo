/*
module futex

import time

fn wait_proc(ft &Futex) {
    ft.wait()
    println("wait done")
}

fn test_1() {
    ft := newFutex()
    ft.wake()
    ft.wake()
    println("first wake???")
    go wait_proc(ft)
    time.sleep(3)
    ft.wake()
    time.sleep(1)
}

*/
