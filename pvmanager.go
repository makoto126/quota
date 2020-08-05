package main

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"time"

	"github.com/shirou/gopsutil/disk"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	typev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	listerv1 "k8s.io/client-go/listers/core/v1"
)

const (
	labelKey = "node"
)

var (
	//ListDuration is the list duration of PV
	ListDuration time.Duration
	//AvailableNum is the number of available PV
	AvailableNum int
	//StorageCapacity is the storage capacity of data disk, if not set, auto detect
	StorageCapacity string
)

type (
	pvManager struct {
		pvCli      typev1.PersistentVolumeInterface
		pvLister   listerv1.PersistentVolumeLister
		dirManager *dirManager
	}

	dirManager struct {
		latest int
	}
)

func newDirManager() (*dirManager, error) {

	files, err := ioutil.ReadDir(BaseDir)
	if err != nil {
		return nil, err
	}

	latest := 0
	for _, f := range files {
		if !f.IsDir() {
			log.Warnln(f.Name, "should not in", BaseDir)
			continue
		}
		n, err := strconv.Atoi(f.Name())
		if err != nil {
			log.Warnln(f.Name, "should not in", BaseDir)
			continue
		}
		if n > latest {
			latest = n
		}
	}

	return &dirManager{
		latest: latest,
	}, nil
}

func (dm *dirManager) Clean(num string) error {

	target := path.Join(BaseDir, num)
	dir, err := ioutil.ReadDir(target)
	if err != nil {
		return err
	}

	for _, d := range dir {
		if err := os.RemoveAll(path.Join(target, d.Name())); err != nil {
			return err
		}
	}

	return nil
}

func (dm *dirManager) AddDir() (int, error) {

	target := path.Join(BaseDir, strconv.Itoa(dm.latest+1))
	err := os.Mkdir(target, os.FileMode(0755))
	if err != nil {
		return -1, err
	}

	dm.latest++

	err = setProjid(target, strconv.Itoa(dm.latest))

	return dm.latest, err
}

func (dm *dirManager) Withdraw() error {

	err := os.Remove(path.Join(BaseDir, strconv.Itoa(dm.latest)))
	if err != nil {
		return err
	}
	dm.latest--
	return nil
}

func newPvManager(
	pvCli typev1.PersistentVolumeInterface,
	pvLister listerv1.PersistentVolumeLister,
) (*pvManager, error) {

	dirManager, err := newDirManager()
	if err != nil {
		return nil, err
	}

	if StorageCapacity == "" {
		sc, err := getCapacity(BaseDir)
		if err != nil {
			log.Fatal(err.Error())
		}
		StorageCapacity = sc
	}

	return &pvManager{
		pvCli:      pvCli,
		pvLister:   pvLister,
		dirManager: dirManager,
	}, nil
}

func (pm *pvManager) Run() {

	selector := labels.SelectorFromSet(labels.Set{labelKey: NodeName})

	for range time.Tick(ListDuration) {

		pvs, err := pm.pvLister.List(selector)
		if err != nil {
			log.Errorln(err)
		}

		available := 0
		for _, pv := range pvs {

			switch pv.Status.Phase {
			case corev1.VolumeReleased:
				_, num := path.Split(pv.Spec.Local.Path)
				if err := pm.dirManager.Clean(num); err != nil {
					log.Errorln(err)
					continue
				}

				if err := pm.reuse(pv); err != nil {
					log.Errorln(err)
					continue
				}

				available++
			case corev1.VolumeAvailable:
				available++
			case corev1.VolumeFailed:
				log.Warnln(pv.Name, corev1.VolumeFailed)
			default:
			}
		}

		if available > AvailableNum {
			continue
		}

		shouldAdd := AvailableNum - available
		for i := 0; i < shouldAdd; i++ {

			latest, err := pm.dirManager.AddDir()
			if err != nil {
				log.Error(err)
			}
			if latest == -1 {
				continue
			}
			if err := pm.create(latest); err != nil {
				log.Error(err)
				if err := pm.dirManager.Withdraw(); err != nil {
					log.Error(err)
				}
			}
		}
	}
}

func (pm *pvManager) reuse(pv *corev1.PersistentVolume) error {

	patchJSON := map[string]interface{}{
		"spec": map[string]interface{}{
			"claimRef": nil,
		},
	}

	patchData, err := json.Marshal(patchJSON)
	if err != nil {
		return err
	}

	_, err = pm.pvCli.Patch(pv.Name, types.StrategicMergePatchType, patchData)
	if err != nil {
		return err
	}

	return nil
}

func (pm *pvManager) create(latest int) error {

	latestStr := strconv.Itoa(latest)
	pv := new(corev1.PersistentVolume)

	pv.SetName(NodeName + "-" + latestStr)
	pv.SetLabels(map[string]string{labelKey: NodeName})

	pv.Spec.Capacity = corev1.ResourceList{
		"storage": resource.MustParse(StorageCapacity),
	}
	volumeMode := corev1.PersistentVolumeFilesystem
	pv.Spec.VolumeMode = &volumeMode
	pv.Spec.AccessModes = []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce}
	//Although the RecliamPolicy is Retain, but actually is Recycle(clean and reuse)
	pv.Spec.PersistentVolumeReclaimPolicy = corev1.PersistentVolumeReclaimRetain
	pv.Spec.StorageClassName = StorageClassName
	pv.Spec.Local = &corev1.LocalVolumeSource{
		Path: path.Join(BaseDir, latestStr),
	}
	pv.Spec.NodeAffinity = &corev1.VolumeNodeAffinity{
		Required: &corev1.NodeSelector{
			NodeSelectorTerms: []corev1.NodeSelectorTerm{
				{
					MatchExpressions: []corev1.NodeSelectorRequirement{
						{
							Key:      "kubernetes.io/hostname",
							Operator: corev1.NodeSelectorOpIn,
							Values:   []string{NodeName},
						},
					},
				},
			},
		},
	}

	_, err := pm.pvCli.Create(pv)
	return err
}

func getCapacity(baseDir string) (string, error) {

	us, err := disk.Usage(baseDir)
	if err != nil {
		return "", err
	}

	t := us.Total >> 30
	return strconv.FormatUint(t, 10) + "Gi", nil
}
