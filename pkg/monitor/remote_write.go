package monitor

import (
	"bytes"
	"net/http"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/schidstorm/ulogd_monitor/pkg/packet"

	prompbmarshal "github.com/VictoriaMetrics/VictoriaMetrics/lib/prompb"
	"github.com/golang/snappy"
)

func RemoteWrite(remoteWriteUrl string, metricers []packet.Metricer, srcHostname string) {
	serialized := serializeMetrics(metricers, srcHostname)
	remoteWriteData(remoteWriteUrl, serialized)
}

func serializeMetrics(metricers []packet.Metricer, srcHostname string) []byte {
	timeseries := make([]prompbmarshal.TimeSeries, 0, len(metricers))
	for _, m := range metricers {
		metric := m.ToMetric()

		labels := []prompbmarshal.Label{
			{Name: "__name__", Value: metric.Name},
			{Name: "hostname", Value: srcHostname},
		}
		for k, v := range metric.Labels {
			labels = append(labels, prompbmarshal.Label{Name: k, Value: v})
		}

		sample := prompbmarshal.Sample{
			Value:     metric.Value,
			Timestamp: metric.Time.UnixMilli(),
		}

		ts := prompbmarshal.TimeSeries{
			Labels:  labels,
			Samples: []prompbmarshal.Sample{sample},
		}

		timeseries = append(timeseries, ts)
	}

	req := prompbmarshal.WriteRequest{
		Timeseries: timeseries,
	}

	data := req.MarshalProtobuf(nil)
	return snappy.Encode(nil, data)
}

var deferredLog = &deferredLogger{}

func remoteWriteData(remoteWriteUrl string, data []byte) {
	log := deferredLog.Get()

	httpReq, err := http.NewRequest("POST", remoteWriteUrl, bytes.NewReader(data))
	if err != nil {
		log.Warn().Err(err).Msg("failed to create http request")
	}
	httpReq.Header.Set("Content-Type", "application/x-protobuf")
	httpReq.Header.Set("Content-Encoding", "snappy")
	httpReq.Header.Set("X-Prometheus-Remote-Write-Version", "0.1.0")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		log.Warn().Err(err).Msg("failed to send remote_write request")
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		log.Warn().Int("status_code", resp.StatusCode).Msg("remote_write request failed")
	}
}

type deferredLogger struct {
	lastTime time.Time
}

func (dl *deferredLogger) Get() zerolog.Logger {
	now := time.Now()
	if now.Sub(dl.lastTime) < time.Second*10 {
		return zerolog.Nop()
	}
	dl.lastTime = now
	return log.Logger
}
