package main

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
)

var (
	//RecordDuration is the record duration of metrics
	RecordDuration time.Duration
)

var (
	persistentVolumeUsedKBytes = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "persistentvolume_used_kbytes",
	}, []string{"node", "id"})

	persistentVolumeQuotaKBytes = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "persistentvolume_quota_kbytes",
	}, []string{"node", "id"})
)

func recordMetrics() {
	go func() {

		for range time.Tick(RecordDuration) {
			reports, err := getReport()
			if err != nil {
				log.Errorln(err)
			}
			for _, report := range reports {
				persistentVolumeUsedKBytes.
					WithLabelValues(NodeName, report.Projid).
					Set(float64(report.Used))

				persistentVolumeQuotaKBytes.
					WithLabelValues(NodeName, report.Projid).
					Set(float64(report.Quota))
			}
		}
	}()
}

// ServeMetrics run the prometheus http server
func ServeMetrics() {
	recordMetrics()

	r := gin.Default()
	r.GET("metrics", gin.WrapH(promhttp.Handler()))
	r.Run()
}
