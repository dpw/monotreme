gofiles=$(shell find -name . "*.go")
pkg=github.com/dpw/monotreme

get_vendor_submodules=@if [ -z "$$(find vendor -type f -print -quit)" ] ; then git submodule update --init ; fi

.PHONY: test
test:
	$(get_vendor_submodules)
	rm -rf build/src/$(pkg)
	mkdir -p $(dir build/src/$(pkg))
	ln -s $$PWD build/src/$(pkg)
	GOPATH=$$PWD/build ; GO15VENDOREXPERIMENT=1 ; export GOPATH GO15VENDOREXPERIMENT ; cd build/src/$(pkg) && go test $$(go list ./... | grep -v /vendor/)

.PHONY: cover
cover:
	rm -rf cover
	$(get_vendor_submodules)
	GOPATH=$$PWD/build ; GO15VENDOREXPERIMENT=1 ; export GOPATH GO15VENDOREXPERIMENT ; set -e ; for d in $$(find * -path vendor -prune -o -name "*_test.go" -printf "%h\n" | sort -u); do \
	        if [ "$$d" = "." ] ; then \
			mkdir cover ; \
			p=$(pkg) ; \
			d=cover ; \
		else \
			mkdir -p cover/$$d ; \
			p=$(pkg)/$$d ; \
		fi ; \
	        go test -coverprofile=cover/$$d.out $$p ; \
	        go tool cover -html=cover/$$d.out -o cover/$$d.html ; \
	    done
