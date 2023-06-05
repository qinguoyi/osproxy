.PHONY: all build swager clean help

BINARY="storage"
GO_CMD = $(shell which go)
GO_BUILD_CMD=$(GO_CMD) build
BASEDIR=$(shell pwd)
CONFIG_DIR=conf
EXE=$(BASEDIR)/bin/$(BINARY)

.PHONY: all
all: swager build

.PHONY: build
build:
	@cd ./cmd && CGO_ENABLED=0 $(GO_BUILD_CMD) -v -o $(BASEDIR)/bin/$(BINARY)
	@cp -r ${BASEDIR}/${CONFIG_DIR} $(BASEDIR)/bin/

.PHONY:swager
swager:
	@$(GO_CMD) install github.com/swaggo/swag/cmd/swag@v1.8.1
	@swag fmt -g cmd/main.go -d ./
	@swag init -g cmd/main.go

.PHONY: clean
clean:
	@if [ -f ${EXE} ] ; then rm ${EXE} ; fi

.PHONY: help
help:
	@echo "make all- 编译生成二进制文件, 生成接口文档"
	@echo "make build - 编译 Go 代码, 生成二进制文件"
	@echo "make swag - 生成接口文档"
	@echo "make clean - 移除二进制文件"