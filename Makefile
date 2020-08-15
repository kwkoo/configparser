PREFIX=github.com/kwkoo
PACKAGE=configparser

COVERAGEOUTPUT=coverage.out
COVERAGEHTML=coverage.html

.PHONY: test clean coverage

test:
	@go test $(PREFIX)/$(PACKAGE) -v

clean:
	@rm -f $(COVERAGEOUTPUT) $(COVERAGEHTML)

coverage:
	@go test $(PREFIX)/$(PACKAGE) -cover -coverprofile=$(COVERAGEOUTPUT)
	@go tool cover -html=$(COVERAGEOUTPUT) -o $(COVERAGEHTML)
	open $(COVERAGEHTML)