all: build


build:
	GOARCH=arm64 GOOS=linux CC=aarch64-linux-gnu-gcc CGO_ENABLED=1 go build -a -o ulogd_udp_json_exporter_arm64 . && \
	GOARCH=amd64 GOOS=linux                          CGO_ENABLED=1 go build -a -o ulogd_udp_json_exporter_amd64 .