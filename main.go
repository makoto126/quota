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
	// NodeName is the hostname
	NodeName string
	// StorageClassName to manage
	StorageClassName string
)

// Config by env
type Config struct {
	NodeName         string        `required:"true" split_words:"true"`
	BaseDir          string        `default:"/data" split_words:"true"`
	AvailableNum     int           `default:"1" split_words:"true"`
	DefaultResync    time.Duration `default:"30s" split_words:"true"`
	ListDuration     time.Duration `default:"5s" split_words:"true"`
	StorageCapacity  string        `default:"1000Gi" split_words:"true"`
	StorageClassName string        `default:"local-storage" split_words:"true"`
}

func main() {

	var c Config
	err := envconfig.Process("", &c)
	if err != nil {
		log.Fatal(err.Error())
	}

	NodeName = c.NodeName
	StorageClassName = c.StorageClassName

	//BaseDir should be the mount point of xfs_quota command
	MntPoint = c.BaseDir

	config, err := rest.InClusterConfig()
	if err != nil {
		log.Fatalln(err)
	}
	cli, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalln(err)
	}

	factory := informers.NewSharedInformerFactory(cli, c.DefaultResync)
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
		c.BaseDir,
		c.AvailableNum,
		c.ListDuration,
		c.StorageCapacity,
	)
	if err != nil {
		log.Fatalln(err)
	}
	go pvManager.Run()

	<-stopCh
}
