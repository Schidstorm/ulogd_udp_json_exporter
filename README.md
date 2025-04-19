# ulogd_udp_json_exporter

A simple Prometheus exporter that uses `libnetfilter-log` to read network logs and exposes various network traffic metrics.

> Designed for logging and monitoring network packet drops (e.g., via `nftables`), with rich, customizable metrics for observability.

---

## 📦 Features

- Reads messages from `libnetfilter-log`
- Parses common netfilter fields (`src_ip`, `dest_port`, `ip.protocol`, etc.)
- Exposes metrics via HTTP in Prometheus format

---

## 📊 Exposed Metrics

| Metric Name                             | Labels                        | Description                                           |
|----------------------------------------|-------------------------------|-------------------------------------------------------|
| `ulogd_packets_total`                  | `prefix`                      | Total number of received (blocked) packets            |
| `ulogd_packets_by_protocol_total`      | `prefix`, `protocol`          | Count of packets grouped by IP protocol              |
| `ulogd_packets_by_interface_total`     | `prefix`, `iif`, `oif`        | Count of packets per incoming and outgoing interface |
| `ulogd_packets_by_dest_port_total`     | `prefix`, `port`              | Count of packets grouped by destination port         |
| `ulogd_packets_by_src_ip_total`        | `prefix`, `src_ip`            | Count of packets grouped by source IP                |
| `ulogd_packets_by_dest_ip_total`       | `prefix`, `dest_ip`           | Count of packets grouped by destination IP           |
| `ulogd_packet_size_bytes`              | `prefix` (Histogram)          | Distribution of packet sizes in bytes                |
| `ulogd_json_parse_errors_total`        | —                             | Number of malformed or failed JSON log entries       |
| `ulogd_packet_read_errors_total`       | —                             | Number of times reading from the UDP socket failed   |

---

## 🚀 Usage

### 🛠️ Run

```bash
go run main.go --group 1 --metrics :8080
```

- `--group`: Log group to read from
- `--metrics`: HTTP address to expose Prometheus metrics

---

## 🔥 Example nftables Rule

To log and drop packets (e.g., SSH brute-force attempts):

```nft
log prefix "reject" group 1 drop
```

This logs messages via NFLOG group `1` and prefix `reject`.

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

- [libnetfilter-log](https://www.netfilter.org/projects/libnetfilter_log)
- [Prometheus Go client](https://github.com/prometheus/client_golang)
- [Cobra CLI](https://github.com/spf13/cobra)
- [Zerolog](https://github.com/rs/zerolog)