.SILENT :

export GO111MODULE=on

# Base package
BASE_PACKAGE=github.com/x0rzkov

# App name
APPNAME=golang-nng-surveyor-pattern-demo

# Go configuration
GOOS?=$(shell go env GOHOSTOS)
GOARCH?=$(shell go env GOHOSTARCH)

# Check the local operatig system
is_windows:=$(filter windows,$(GOOS))
is_darwin:=$(filter darwin,$(GOOS))
is_linux:=$(filter linux,$(GOOS))

# Add exe extension if windows target
EXT:=$(if $(is_windows),".exe","")
LDLAGS_LAUNCHER:=$(if $(is_windows),-ldflags "-H=windowsgui",)

# Archive name
ARCHIVE=$(APPNAME)-$(GOOS)-$(GOARCH).tgz

# Main executable name
MAIN_EXE=$(APPNAME)$(EXT)

# CLI executable name
CLI_EXE=$(APPNAME)-ctl$(EXT)

# Launcher filename
LAUNCHER_EXE=$(APPNAME)-launcher$(EXT)

# Plugin name
PLUGIN?=nng

# Plugin filename
PLUGIN_SO=$(APPNAME)-$(PLUGIN).so

# Extract version infos
PKG_VERSION:=github.com/x0rzkov/$(APPNAME)/v2/pkg/version
VERSION:=`git describe --tags --always`
GIT_COMMIT:=`git rev-list -1 HEAD --abbrev-commit`
BUILT:=`date`
define LDFLAGS
-X '$(PKG_VERSION).Version=$(VERSION)' \
-X '$(PKG_VERSION).GitCommit=$(GIT_COMMIT)' \
-X '$(PKG_VERSION).Built=$(BUILT)'
endef

all: build

# Include common Make tasks
root_dir:=$(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))
makefiles:=$(root_dir)/makefiles
include $(makefiles)/help.Makefile

## Clean built files
clean:
	echo ">>> Removing generated files ..."
	# -rm -rf release autogen var/assets/ui pkg/assets/statik.go
.PHONY: clean

## Dependencies
deps:
	export GO111MODULE=off
	echo ">>> Installing dependencies"
	go get github.com/mattn/goreman
	# go get -u github.com/go-bindata/go-bindata/...
	export GO111MODULE=on
.PHONY: deps

## Run code generation
autogen:
	echo ">>> Generating code ..."
	package 
	# goagen bootstrap -o autogen -d $(BASE_PACKAGE)/$(APPNAME)/v3/design
	# echo ">>> Moving Swagger files to assets ..."
	# cp -f $(root_dir)/autogen/swagger/** $(root_dir)/var/assets/

## Build executable
build:
	-mkdir -p bin
	echo ">>> Building: $(MAIN_EXE) $(VERSION) for $(GOOS)-$(GOARCH) ..."
	GOOS=$(GOOS) GOARCH=$(GOARCH) go build -ldflags "$(LDFLAGS)" -o bin/$(MAIN_EXE)
.PHONY: build

release/$(MAIN_EXE): build

## Run tests
test:
	-golint pkg/...
	go test `go list ./... | grep -v autogen`
.PHONY: test

## Install executable
install: release/$(MAIN_EXE)
	echo ">>> Installing $(MAIN_EXE) to ${HOME}/.local/bin/$(MAIN_EXE) ..."
	cp bin/$(MAIN_EXE) ${HOME}/.local/bin/$(MAIN_EXE)
.PHONY: install

## Create Docker image
image:
	echo ">>> Building Docker image ..."
	docker build --rm -t x0rzkov/$(APPNAME) .
.PHONY: image

## Generate changelog
changelog:
	standard-changelog --first-release
.PHONY: changelog

## Create archive
archive:
	echo ">>> Creating release/$(ARCHIVE) archive..."
	tar czf release/$(ARCHIVE) \
		--exclude=*.tgz \
	 	README.md \
		LICENSE \
		CHANGELOG.md \
		-C release/ $(subst release/,,$(wildcard release/*))
	find bin/ -type f -not -name '*.tgz' -delete
.PHONY: archive

## Create distribution binaries
distribution:
	GOOS=linux GOARCH=amd64 make build plugins archive
	GOOS=linux GOARCH=arm64 make build archive
	GOOS=linux GOARCH=arm make build archive
	GOOS=windows make build archive
	GOOS=darwin make build archive
.PHONY: distribution

## Bulid plugin (defined by PLUGIN variable)
plugin:
	-mkdir -p shared/plugins
	-if [ -d "plugins/$(PLUGIN)/views" ]; then cd plugins/$(PLUGIN) && go-bindata views/... ; fi
	echo ">>> Building: $(PLUGIN_SO) $(VERSION) for $(GOOS)-$(GOARCH) ..."
	cd plugins/$(PLUGIN) && GOOS=$(GOOS) GOARCH=$(GOARCH) go build -buildmode=plugin -o ../../shared/plugins/$(PLUGIN_SO)
.PHONY: plugin

## Build all plugins
plugins: 
	-rm -rf ./shared/plugins/*.so
	#GOARCH=amd64 PLUGIN=messages-log make plugin	
	#GOARCH=amd64 PLUGIN=messages-save make plugin
	#GOARCH=amd64 PLUGIN=paper2code make plugin
.PHONY: plugins
 
