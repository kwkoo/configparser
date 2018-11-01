PREFIX=github.com/kwkoo
PACKAGE=argparser

GOPATH:=$(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))
GOBIN=$(GOPATH)/bin
COVERAGEOUTPUT=coverage.out
COVERAGEHTML=coverage.html

.PHONY: test clean coverage

test:
	@GOPATH=$(GOPATH) GOBIN=$(GOBIN) go test $(PREFIX)/$(PACKAGE) -v

clean:
	rm -f $(GOPATH)/bin/$(PACKAGE) $(GOPATH)/pkg/*/$(PACKAGE).a $(GOPATH)/$(COVERAGEOUTPUT) $(GOPATH)/$(COVERAGEHTML)

coverage:
	@GOPATH=$(GOPATH) GOBIN=$(GOBIN) go test $(PREFIX)/$(PACKAGE) -cover -coverprofile=$(GOPATH)/$(COVERAGEOUTPUT)
	@GOPATH=$(GOPATH) GOBIN=$(GOBIN) go tool cover -html=$(GOPATH)/$(COVERAGEOUTPUT) -o $(GOPATH)/$(COVERAGEHTML)
	open $(GOPATH)/$(COVERAGEHTML)