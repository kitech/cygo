
corona goroutine implement for Vlang.

Note: Linux only

### Install

```
cd ~/code
git clone https://github.com/kitech/corona.git
cd corona
git clone https://github.com/ivmai/bdwgc.git
git clone https://github.com/srdja/Collections-C.git cltc

cd bdwgc
./autogen.sh
./configure
make
cd ..

cd cltc
# if have cpptest problem, edit CMakeLists.txt, comment enable_testing and add_subdirectory(test) 
cmake -DCMAKE_INSTALL_PREFIX=$PWD .
make && make install
cd ..

make -C corona/corona-c lcrn

```

To install `libevent`, just use linux package system.

Last, put corona-v in vlib dir:

```
cp -a corona/corona-v v/corona
# edit v/corona/corona.v, change path to home dir
```

### Examples

test.v
```
module main

import corona

fn main() {
  crn := corona.new()
  crn.post(test, crn)
  corona.sleep(3)
}

fn test(arg voidptr) {
   id := corona.goid()
   println('in test goid=$id')
}

```
