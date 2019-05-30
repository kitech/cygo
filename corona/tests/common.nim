import times

proc usleepc(usec:int) : int {.importc:"usleep".}

proc nowt0() : DateTime = times.fromUnix(epochTime().int64).utc()
proc nowt1() : int64 = epochTime().int64
