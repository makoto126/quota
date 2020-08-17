package main

import (
	"path"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/shirou/gopsutil/disk"
	log "github.com/sirupsen/logrus"
)

var (
	//RecordDuration is the record duration of metrics
	RecordDuration time.Duration
)

//prometheus metrics
var (
	persistentVolumeUsedKBytes = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "persistentvolume_used_kbytes",
	}, []string{"node", "id"})

	persistentVolumeQuotaKBytes = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "persistentvolume_quota_kbytes",
	}, []string{"node", "id"})

	dataDiskReadCount = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "data_disk_read_count",
	}, []string{"node", "name"})

	dataDiskWriteCount = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "data_disk_write_count",
	}, []string{"node", "name"})

	dataDiskReadBytes = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "data_disk_read_bytes",
	}, []string{"node", "name"})

	dataDiskWriteBytes = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "data_disk_write_bytes",
	}, []string{"node", "name"})

	dataDiskReadTime = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "data_disk_read_time",
	}, []string{"node", "name"})

	dataDiskWriteTime = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "data_disk_write_time",
	}, []string{"node", "name"})
)

func recordMetrics() {

	var diskName string

	partitions, err := disk.Partitions(false)
	if err != nil {
		log.Errorln(err)
	}
	for _, p := range partitions {
		if p.Mountpoint == BaseDir {
			_, diskName = path.Split(p.Device)
		}
	}

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

			ioCounters, err := disk.IOCounters(diskName)
			if err != nil {
				log.Errorln(err)
			}
			stat := ioCounters[diskName]
			dataDiskReadCount.
				WithLabelValues(NodeName, diskName).
				Set(float64(stat.ReadCount))
			dataDiskWriteCount.
				WithLabelValues(NodeName, diskName).
				Set(float64(stat.WriteCount))
			dataDiskReadBytes.
				WithLabelValues(NodeName, diskName).
				Set(float64(stat.ReadBytes))
			dataDiskWriteBytes.
				WithLabelValues(NodeName, diskName).
				Set(float64(stat.WriteBytes))
			dataDiskReadTime.
				WithLabelValues(NodeName, diskName).
				Set(float64(stat.ReadTime))
			dataDiskWriteTime.
				WithLabelValues(NodeName, diskName).
				Set(float64(stat.WriteTime))
		}
	}()
}

// ServeMetrics run the prometheus http server
func ServeMetrics() {

	recordMetrics()

	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()
	r.GET("metrics", gin.WrapH(promhttp.Handler()))
	r.Run()
}
