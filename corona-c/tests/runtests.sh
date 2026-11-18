#!/bin/bash
# runtests.sh — build & run the corona-c test suite
# usage: ./runtests.sh [test-name ...]   (default: all)
#        VALGRIND=1 ./runtests.sh        (run under valgrind)
#        RETRIES=n ./runtests.sh         (per-test retries, default 2)
cd "$(dirname "$0")"
make -j4 || exit 1

tests="$@"
if [ -z "$tests" ]; then
  tests=$(ls t*_*.c 2>/dev/null | grep -v t00_ | sed 's/\.c$//')
fi
retries=${RETRIES:-2}

fail=0
for t in $tests; do
  echo "=== $t ==="
  rc=1
  attempt=0
  while [ $attempt -le $retries ]; do
    if [ "$VALGRIND" = "1" ]; then
      timeout 60 valgrind --error-exitcode=99 --leak-check=full \
        --log-file=/tmp/opencode/$t.vg ./$t > /tmp/opencode/$t.log 2>&1
    else
      timeout 60 ./$t > /tmp/opencode/$t.log 2>&1
    fi
    rc=$?
    [ $rc -eq 0 ] && break
    attempt=$((attempt+1))
    if [ $attempt -le $retries ]; then
      echo "RETRY $t rc=$rc (attempt $attempt)"
    fi
  done
  if [ $rc -ne 0 ]; then
    echo "FAIL $t rc=$rc (log: /tmp/opencode/$t.log)"
    tail -5 /tmp/opencode/$t.log
    fail=1
  else
    echo "PASS $t"
  fi
done
exit $fail
