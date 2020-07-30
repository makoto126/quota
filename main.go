package main

import (
	"time"

	"github.com/kelseyhightower/envconfig"
	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	defaultResync = 30 * time.Second
)

// Config by env
type Config struct {
	NodeName      string        `required:"true"`
	BaseDir       string        `default:"/data" split_words:"true"`
	AvailableNum  int           `default:"1" split_words:"true"`
	DefaultResync time.Duration `default:"30s" split_words:"true"`
	ListDuration  time.Duration `default:"5s" split_words:"true"`
	Storage       string        `default:"1000Gi" split_words:"true"`
}

func main() {

	var c Config
	err := envconfig.Process("", &c)
	if err != nil {
		log.Fatal(err.Error())
	}

	// creates the in-cluster config
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

	handler := newQuotaHandler()
	pvcInformer.AddEventHandler(handler)

	stopCh := make(chan struct{})

	log.Println("Start SharedInformerFactory...")
	factory.Start(stopCh)

	pvManager, err := newPvManager(
		c.NodeName,
		cli.CoreV1().PersistentVolumes(),
		pvLister,
		c.BaseDir,
		c.AvailableNum,
		c.ListDuration,
		c.Storage,
	)
	if err != nil {
		log.Fatalln(err)
	}
	pvManager.Run()

	<-stopCh
}
