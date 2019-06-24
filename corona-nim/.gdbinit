# note: ~/.gdbinit

handle SIGXCPU SIG33 SIG35 SIGPWR nostop noprint
set pagination off
#set print thread-events off
file ./corona
r
#thread apply all bt full
thread apply all bt
quit
