all: gr1

gr0:
	nim --threads:on --listcmd c goro.nim

gr1:
	#-d:noSignalHandler
	nim -d:noSignalHandler --stacktrace:on --threads:on --gc:boehm  --listcmd \
		-p:${HOME}/.nimble/mulib c corona.nim

# some tricks
# gdb: handle SIGXCPU SIG33 SIG35 SIGPWR nostop noprint
