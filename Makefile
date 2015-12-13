pkg=github.com/dpw/monotreme

get_vendor_submodules=@if [ -z "$$(find vendor -type f -print -quit)" ] ; then git submodule update --init ; fi

goprep:=GOPATH=$$PWD/build ; GO15VENDOREXPERIMENT=1 ; export GOPATH GO15VENDOREXPERIMENT ; rm -rf build/src/$(pkg) && mkdir -p $(dir build/src/$(pkg)) && ln -s $$PWD build/src/$(pkg) &&

build/bin/node: $(shell find -name vendor -prune -o -name "*.go" -print)
	$(get_vendor_submodules)
	$(goprep) cd build/src/$(pkg)/cmd && go install ./...

.PHONY: test
test:
	$(get_vendor_submodules)
	$(goprep) cd build/src/$(pkg) && go test $$(go list ./... | grep -v /vendor/)

.PHONY: cover
cover:
	rm -rf cover
	$(get_vendor_submodules)
	$(goprep) set -e ; for d in $$(find * -path vendor -prune -o -name "*_test.go" -printf "%h\n" | sort -u); do \
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

.PHONY: clean
clean::
	rm -rf cover build

