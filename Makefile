gofiles=$(shell find -name . "*.go")
pkg=github.com/dpw/osedax

get_vendor_submodules=@if [ -z "$$(find vendor -type f -print -quit)" ] ; then git submodule update --init ; fi

.PHONY: test
test:
	$(get_vendor_submodules)
	rm -rf build/src/$(pkg)
	mkdir -p $(dir build/src/$(pkg))
	ln -s $$PWD build/src/$(pkg)
	GOPATH=$$PWD/build ; GO15VENDOREXPERIMENT=1 ; export GOPATH GO15VENDOREXPERIMENT ; cd build/src/$(pkg) && go test $$(go list ./... | grep -v /vendor/)
