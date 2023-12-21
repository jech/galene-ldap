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
	# Its calling everything in the makefile ...
	@echo ""
	@echo "CI BUILD starting ..."
	$(MAKE) print 
	$(MAKE) clean-all
	$(MAKE) build-debug
	$(MAKE) data-bootstrap
	$(MAKE) run-debug
	@echo ""
	@echo "CI BUILD ended ...."

clean-all: data-clean build-clean

upgrade:
	# https://github.com/oligot/go-mod-upgrade
	# https://github.com/oligot/go-mod-upgrade/releases/tag/v0.9.1
	go install github.com/oligot/go-mod-upgrade@v0.9.1
	go-mod-upgrade
	go mod tidy

build-init:
	mkdir -p $(BIN_ROOT)
build-clean:
	rm -rf $(BIN_ROOT)
build: build-init
	CGO_ENABLED=1 go build -o $(BIN) .
build-debug: build-init
	CGO_ENABLED=1 go build -ldflags='-s -w' -o $(BIN) .


data-init:
	mkdir -p $(DATA_ROOT)
data-clean:
	rm -rf $(DATA_ROOT)

# toggle to choose what example you want to use...
#DATA_EXAMPLE=examples/01/galene-ldap.json
DATA_EXAMPLE=examples/02/galene-ldap.json
data-bootstrap: data-init
	cp $(DATA_EXAMPLE) $(DATA)

	# TODO: cp in certs...Gen with mkcert.

run-h:
	$(BIN) -h
run:
	$(BIN) -data $(DATA_ROOT)
	# http://localhost:8088
run-debug:
	nohup $(BIN) -debug -data $(DATA_ROOT) &
	# http://localhost:8088

