gofiles=$(shell find -name . "*.go")
pkg=github.com/dpw/osedax

test:
	rm -rf build/src/$(pkg)
	mkdir -p build/src/$(pkg)
	cp -a $$(find * -path "build" -prune -o -name "*.go" -printf "%H\n" | sort -u) build/src/$(pkg)
	GOPATH=$$PWD/build ; export GOPATH ; cd build/src/$(pkg) \
		&& go get -t ./... && go test ./...
