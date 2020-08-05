package main

import (
	"time"

	"github.com/kelseyhightower/envconfig"
	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var (
	//NodeName is the hostname of Host（not Docker）
	NodeName string
	//BaseDir is the mount point of data disk
	BaseDir string
	//StorageClassName of PV and PVC
	StorageClassName string
	//DefaultResync is the default resync duration of imformer
	DefaultResync time.Duration
)

// Config by env
type Config struct {
	NodeName         string        `required:"true" split_words:"true"`
	BaseDir          string        `default:"/data" split_words:"true"`
	AvailableNum     int           `default:"1" split_words:"true"`
	DefaultResync    time.Duration `default:"30s" split_words:"true"`
	ListDuration     time.Duration `default:"5s" split_words:"true"`
	StorageClassName string        `default:"local-storage" split_words:"true"`
	StorageCapacity  string        `split_words:"true"`
	RecordDuration   time.Duration `default:"30s" split_words:"true"`
}

func main() {

	var c Config
	err := envconfig.Process("", &c)
	if err != nil {
		log.Fatal(err.Error())
	}

	NodeName = c.NodeName
	BaseDir = c.BaseDir
	StorageClassName = c.StorageClassName
	DefaultResync = c.DefaultResync
	ListDuration = c.ListDuration
	AvailableNum = c.AvailableNum
	StorageCapacity = c.StorageCapacity
	RecordDuration = c.RecordDuration

	config, err := rest.InClusterConfig()
	if err != nil {
		log.Fatalln(err)
	}
	cli, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalln(err)
	}

	factory := informers.NewSharedInformerFactory(cli, DefaultResync)
	pvcInformer := factory.Core().V1().PersistentVolumeClaims().Informer()
	pvLister := factory.Core().V1().PersistentVolumes().Lister()

	handler := &quotaHandler{
		corev1Cli: cli.CoreV1(),
		pvLister:  pvLister,
	}
	pvcInformer.AddEventHandler(handler)

	stopCh := make(chan struct{})

	log.Println("Start SharedInformerFactory...")
	factory.Start(stopCh)

	pvManager, err := newPvManager(
		cli.CoreV1().PersistentVolumes(),
		pvLister,
	)
	if err != nil {
		log.Fatalln(err)
	}
	go pvManager.Run()
	go ServeMetrics()

	<-stopCh
}
