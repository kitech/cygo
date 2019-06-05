all: gr1

gr0:
	nim --threads:on --listcmd c goro.nim

gr1:
	#-d:noSignalHandler
	#--listcmd
	nim -d:noSignalHandler --stacktrace:on --threads:on --gc:boehm \
		-p:${HOME}/.nimble/mulib c corona.nim

dc:
	nim doc -p:~/.nimble/mulib/ corona.nim

# some tricks
# gdb: handle SIGXCPU SIG33 SIG35 SIGPWR nostop noprint
