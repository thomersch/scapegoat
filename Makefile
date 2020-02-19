export CGO_CFLAGS=-I. -I/usr/local/include
export CGO_LDFLAGS=-L/usr/local/lib

build:
	go build
