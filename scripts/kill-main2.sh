set -x
ps axu|grep -aiE '(corona2|main2)'|awk '{print $2}'|xargs kill -9
