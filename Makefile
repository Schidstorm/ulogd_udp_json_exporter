project = ulogd_udp_json_exporter
arch = $(shell go env GOARCH)

all: test build

test:
	go test -v ./...

build: build_arm64 build_arch

build_arm64:
	bash remoteBuild.sh "$(shell cat remotes.txt)"

build_arch: test
	sudo apt-get install -y libnetfilter-log-dev && \
	CGO_ENABLED=1 go build -a -o ulogd_udp_json_exporter_$(arch) .
	