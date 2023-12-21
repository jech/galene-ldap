OS_GO_BIN_NAME=go
ifeq ($(shell uname),Windows)
	OS_GO_BIN_NAME=go.exe
endif

OS_GO_OS=$(shell $(OS_GO_BIN_NAME) env GOOS)
# toggle to fake being windows..
#OS_GO_OS=windows

BIN_ROOT=$(PWD)/.bin
BIN=$(BIN_ROOT)/galene-ldap
ifeq ($(OS_GO_OS),windows)
	BIN=$(BIN_ROOT)/galene-ldap.exe
endif

DATA_ROOT=$(PWD)/.data
DATA=$(DATA_ROOT)/galene-ldap.json

print:
	@echo ""
	@echo "OS_GO_BIN_NAME:  $(OS_GO_BIN_NAME)"
	@echo ""
	@echo "OS_GO_OS:  $(OS_GO_OS)"
	@echo ""
	@echo "BIN_ROOT:  $(BIN_ROOT)"
	@echo "BIN:       $(BIN)"
	@echo ""
	@echo "DATA_ROOT: $(DATA_ROOT)"
	@echo "DATA:      $(DATA)"
	@echo ""

ci-build: 
	# You can call this locally. github workflow also calls it.
	@echo ""
	@echo "CI BUILD starting ..."
	$(MAKE) print 
	$(MAKE) clean
	$(MAKE) build
	$(MAKE) data-bootstrap
	$(MAKE) run
	@echo ""
	@echo "CI BUILD ended ...."

clean: data-del build-del

upgrade:
	# https://github.com/oligot/go-mod-upgrade
	# https://github.com/oligot/go-mod-upgrade/releases/tag/v0.9.1
	go install github.com/oligot/go-mod-upgrade@v0.9.1
	go-mod-upgrade
	go mod tidy

build:
	CGO_ENABLED=1 go build -o $(BIN) .
build-del:
	rm -rf $(BIN_ROOT)

data:
	mkdir -p $(DATA_ROOT)
data-del:
	rm -rf $(DATA_ROOT)

DATA_EXAMPLE=examples/02/galene-ldap.json
data-bootstrap: data
	# cp in config
	# change to choose what example you want to use...
	cp $(DATA_EXAMPLE) $(DATA)

	# TODO: cp in certs...Gen with mkcert.

run-h:
	$(BIN) -h
run:
	nohup $(BIN) -debug -data $(DATA) &
	# http://localhost:8088

