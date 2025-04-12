all: build


build:
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -a -ldflags '-extldflags "-static"' -o ulogd_udp_json_exporter_arm64 . && \
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -a -ldflags '-extldflags "-static"' -o ulogd_udp_json_exporter_amd64 .