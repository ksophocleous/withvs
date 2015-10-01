withvs
===

command line tool that can launch commands in a visual studio environment without launching vcvars.bat.

This means it can build software that require you to be in a visual studio environment without leaving your bash shell.

install
===

> `go install github.com/ksophocleous/withvs`

usage
===
provided that `$GOPATH/bin` is in your path, you can do `withvs --vs12 -- ${ANY_COMMAND}`

This will launch the visual studio environment (by default it will launch the 64 bit environment but it can be changed by passing the `--32` parameter)

It also removes `mingw` from the `PATH` to prevent any misconfigurations from incorrect configure scripts.
