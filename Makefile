project = ulogd_monitor
arch = $(shell /usr/local/go/bin/go env GOARCH)

all: build


build: tidy build_arm64 build_arch

tidy:
	/usr/local/go/bin/go mod tidy

build_arm64:
	bash remoteBuild.sh "$(shell cat remotes.txt)"

build_arch: 
	sudo apt-get install -y libnetfilter-log-dev && \
	CGO_ENABLED=1 /usr/local/go/bin/go build -v -a -o ulogd_monitor_$(arch) ./cmd/monitor
	
proto:
	find -type f -name '*.proto' | xargs -L1 protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		./pkg/pb/monitor.proto
