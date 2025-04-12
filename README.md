# ulogd_udp_json_exporter

A simple Prometheus exporter that listens for `ulogd2` JSON logs over UDP and exposes various network traffic metrics.

> Designed for logging and monitoring network packet drops (e.g., via `nftables`), with rich, customizable metrics for observability.

---

## 📦 Features

- Listens on a UDP port for JSON-formatted logs from `ulogd2`
- Parses common netfilter fields (`src_ip`, `dest_port`, `ip.protocol`, etc.)
- Exposes metrics via HTTP in Prometheus format

---

## 📊 Exposed Metrics

| Metric Name                             | Labels              | Description                                       |
|----------------------------------------|---------------------|---------------------------------------------------|
| `ulogd_packets_total`                  | —                   | Total number of received (blocked) packets        |
| `ulogd_packets_by_protocol_total`      | `protocol`          | Count of packets grouped by IP protocol           |
| `ulogd_packets_by_interface_total`     | `interface`         | Count of packets per incoming interface           |
| `ulogd_packets_by_dest_port_total`     | `port`              | Count of packets grouped by destination port      |
| `ulogd_packets_by_src_ip_total`        | `src_ip`            | Count of packets grouped by source IP             |
| `ulogd_packet_size_bytes`              | — (Histogram)       | Distribution of packet sizes in bytes             |
| `ulogd_json_parse_errors_total`        | —                   | Number of malformed or failed JSON log entries    |

---

## 🚀 Usage

### 🛠️ Run

```bash
go run main.go --listen :9999 --metrics :8080
```

- `--listen`: UDP address to receive `ulogd2` logs
- `--metrics`: HTTP address to expose Prometheus metrics

---

## 🧪 Example `ulogd.conf`

Here's a working configuration for `ulogd2` with JSON UDP output:

```ini
[global]
logfile="syslog"
loglevel=3

plugin="/usr/lib/aarch64-linux-gnu/ulogd/ulogd_inppkt_NFLOG.so"
plugin="/usr/lib/aarch64-linux-gnu/ulogd/ulogd_filter_IFINDEX.so"
plugin="/usr/lib/aarch64-linux-gnu/ulogd/ulogd_filter_IP2STR.so"
plugin="/usr/lib/aarch64-linux-gnu/ulogd/ulogd_filter_HWHDR.so"
plugin="/usr/lib/aarch64-linux-gnu/ulogd/ulogd_raw2packet_BASE.so"
plugin="/usr/lib/aarch64-linux-gnu/ulogd/ulogd_output_JSON.so"

stack=log2:NFLOG,base1:BASE,ifi1:IFINDEX,ip2str1:IP2STR,mac2str1:HWHDR,json1:JSON

[log2]
group=1

[json1]
sync=1
mode="udp"
host="127.0.0.1"
port="9999"
```

---

## 🔥 Example nftables Rule

To log and drop packets (e.g., SSH brute-force attempts):

```nft
log group 1 drop
```

This sends log messages to `ulogd2` via NFLOG group `1`.

---

## 📥 Prometheus Scrape Config

Example `prometheus.yml` snippet:

```yaml
scrape_configs:
  - job_name: "ulogd_exporter"
    static_configs:
      - targets: ["localhost:8080"]
```

---

## 🙏 Credits

- [ulogd2](https://www.netfilter.org/projects/ulogd/)
- [Prometheus Go client](https://github.com/prometheus/client_golang)
- [Cobra CLI](https://github.com/spf13/cobra)
- [Zerolog](https://github.com/rs/zerolog)