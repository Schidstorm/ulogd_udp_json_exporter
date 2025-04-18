
FROM ubuntu

ARG go_version=1.24.2

RUN apt update
RUN apt-get install -y gcc wget
RUN wget -O go.tar.gz "https://go.dev/dl/go${go_version}.linux-$(dpkg --print-architecture).tar.gz" && \
    rm -rf /usr/local/go && tar -C /usr/local -xzf go.tar.gz && \
    rm -f go.tar.gz
ENV PATH="/usr/local/go/bin:${PATH}"

RUN mkdir -p /build /root/code
WORKDIR /root/code
RUN apt-get install -y libnetfilter-log-dev
COPY go.mod go.sum /root/code/
RUN go mod download
RUN go mod tidy

COPY . /root/code

RUN CGO_ENABLED=1 /usr/local/go/bin/go build -x -o /build/out .