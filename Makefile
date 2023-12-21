

BIN_ROOT=$(PWD)/.bin
BIN=$(BIN_ROOT)/galene-ldap

DATA_ROOT=$(PWD)/.data
DATA=$(DATA_ROOT)/galene-ldap.json

print:
	@echo ""
	@echo "BIN_ROOT:  $(BIN_ROOT)"
	@echo ""
	@echo "DATA_ROOT: $(DATA_ROOT)"
	@echo ""
	@echo ""


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
data-bootstrap: data
	# cp in config
	$(REPO_CMD) cp examples/02/galene-ldap.json $(DATA)
	# cp in certs...

run-h:
	$(BIN) -h
run:
	nohup $(BIN) -debug -data $(DATA) &
	#  http://localhost:8088

