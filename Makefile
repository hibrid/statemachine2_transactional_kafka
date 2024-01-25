BUILD_ID ?= dev
BUILD_TIME=$(shell date -u '+%Y-%m-%dT%H:%M:%S%z')

GOPATH ?= $GOPATH
GO_LD_FLAGS:=-X "github.com/hibrid/statemachine2_transactional_kafka/cmd.version=${BUILD_VERSION}" -X "github.com/hibrid/statemachine2_transactional_kafka/cmd.buildTime=${BUILD_TIME}"

build: 
	packr install -ldflags "${GO_LD_FLAGS}" .