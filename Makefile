project = ulogd_udp_json_exporter

all: test build

test:
	go test -v ./...

build: build_arm64 build_amd64

build_arm64:
	# rsync to scan with user admin to port 60333
	rsync -e 'ssh -p 60333' -avz --delete --exclude=.git --exclude=Makefile --exclude=README.md '--exclude=ulogd_udp_json_exporter_*' . admin@scan:ulogd_udp_json_exporter && \
	ssh -p 60333 admin@scan 'cd ulogd_udp_json_exporter' && \
	ssh -p 60333 admin@scan 'cd ulogd_udp_json_exporter && docker build -t ulogd_udp_json_exporter_arm64 .'' && \
	ssh -p 60333 admin@scan 'cd ulogd_udp_json_exporter && docker create --name ulogd_udp_json_exporter_arm64' && \
	ssh -p 60333 admin@scan 'cd ulogd_udp_json_exporter && docker cp ulogd_udp_json_exporter_arm64:/build/out /tmp/ulogd_udp_json_exporter_arm64' && \
	ssh -p 60333 admin@scan 'cd ulogd_udp_json_exporter && docker rm -f ulogd_udp_json_exporter_arm64' && \

build_amd64:
	GOARCH=amd64 GOOS=linux CGO_ENABLED=1 go build -a -o ulogd_udp_json_exporter_amd64 .
	