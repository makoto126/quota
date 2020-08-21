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
	QuotadPersistentVolumeUsedKBytes = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "quotad_persistentvolume_used_kbytes",
	}, []string{"node", "id"})

	QuotadPersistentVolumeQuotaKBytes = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "quotad_persistentvolume_quota_kbytes",
	}, []string{"node", "id"})

	QuotadDataDiskReadCount = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "quotad_data_disk_read_count",
	}, []string{"node", "name"})

	QuotadDataDiskWriteCount = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "quotad_data_disk_write_count",
	}, []string{"node", "name"})

	QuotadDataDiskReadBytes = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "quotad_data_disk_read_bytes",
	}, []string{"node", "name"})

	QuotadDataDiskWriteBytes = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "quotad_data_disk_write_bytes",
	}, []string{"node", "name"})

	QuotadDataDiskReadTime = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "quotad_data_disk_read_time",
	}, []string{"node", "name"})

	QuotadDataDiskWriteTime = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "quotad_data_disk_write_time",
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
				QuotadPersistentVolumeUsedKBytes.
					WithLabelValues(NodeName, report.Projid).
					Set(float64(report.Used))

				QuotadPersistentVolumeQuotaKBytes.
					WithLabelValues(NodeName, report.Projid).
					Set(float64(report.Quota))
			}

			ioCounters, err := disk.IOCounters(diskName)
			if err != nil {
				log.Errorln(err)
			}
			stat := ioCounters[diskName]
			QuotadDataDiskReadCount.
				WithLabelValues(NodeName, diskName).
				Set(float64(stat.ReadCount))
			QuotadDataDiskWriteCount.
				WithLabelValues(NodeName, diskName).
				Set(float64(stat.WriteCount))
			QuotadDataDiskReadBytes.
				WithLabelValues(NodeName, diskName).
				Set(float64(stat.ReadBytes))
			QuotadDataDiskWriteBytes.
				WithLabelValues(NodeName, diskName).
				Set(float64(stat.WriteBytes))
			QuotadDataDiskReadTime.
				WithLabelValues(NodeName, diskName).
				Set(float64(stat.ReadTime))
			QuotadDataDiskWriteTime.
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
