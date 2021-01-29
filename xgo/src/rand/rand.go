package rand

/*
   #include <time.h>
   #include <stdlib.h>
*/
import "C"

func init() {
	t := C.time(nil)
	C.srand(t)
}
