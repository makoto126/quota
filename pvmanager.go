package main

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"time"

	typev1 "k8s.io/client-go/kubernetes/typed/core/v1"

	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	listerv1 "k8s.io/client-go/listers/core/v1"
)

const (
	labelKey = "node"
)

type (
	pvManager struct {
		pvCli        typev1.PersistentVolumeInterface
		pvLister     listerv1.PersistentVolumeLister
		listDuration time.Duration
		dirManager   *dirManager
		availableNum int
		storage      string
		hostname     string
	}

	dirManager struct {
		baseDir string
		latest  int
	}
)

func newDirManager(baseDir string) (*dirManager, error) {

	files, err := ioutil.ReadDir(baseDir)
	if err != nil {
		return nil, err
	}

	latest := 0
	for _, f := range files {
		if !f.IsDir() {
			log.Warnln(f.Name, "should not in", baseDir)
			continue
		}
		n, err := strconv.Atoi(f.Name())
		if err != nil {
			log.Warnln(f.Name, "should not in", baseDir)
			continue
		}
		if n > latest {
			latest = n
		}
	}

	return &dirManager{
		baseDir: baseDir,
		latest:  latest,
	}, nil
}

func (dm *dirManager) Clean(num string) error {

	target := path.Join(dm.baseDir, num)
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

	err := os.Mkdir(path.Join(dm.baseDir, strconv.Itoa(dm.latest+1)), os.FileMode(0755))
	if err != nil {
		return -1, err
	}

	dm.latest++
	return dm.latest, nil
}

func (dm *dirManager) Withdraw() error {

	err := os.Remove(path.Join(dm.baseDir, strconv.Itoa(dm.latest)))
	if err != nil {
		return err
	}
	dm.latest--
	return nil
}

func newPvManager(
	pvCli typev1.PersistentVolumeInterface,
	pvLister listerv1.PersistentVolumeLister,
	baseDir string,
	availableNum int,
	listDuration time.Duration,
	storage string,
) (*pvManager, error) {

	hostname, err := os.Hostname()
	if err != nil {
		return nil, err
	}

	dirManager, err := newDirManager(baseDir)
	if err != nil {
		return nil, err
	}

	return &pvManager{
		pvCli:        pvCli,
		pvLister:     pvLister,
		listDuration: listDuration,
		dirManager:   dirManager,
		availableNum: availableNum,
		storage:      storage,
		hostname:     hostname,
	}, nil
}

func (pm *pvManager) Run() {

	selector := labels.SelectorFromSet(labels.Set{labelKey: pm.hostname})

	for range time.Tick(pm.listDuration) {

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

		if available > pm.availableNum {
			continue
		}

		shouldAdd := pm.availableNum - available
		for i := 0; i < shouldAdd; i++ {

			latest, err := pm.dirManager.AddDir()
			if err != nil {
				log.Error(err)
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

	pv.SetName(pm.hostname + "-" + latestStr)
	pv.SetLabels(map[string]string{labelKey: pm.hostname})

	pv.Spec.Capacity = corev1.ResourceList{
		"storage": resource.MustParse(pm.storage),
	}
	volumeMode := corev1.PersistentVolumeFilesystem
	pv.Spec.VolumeMode = &volumeMode
	pv.Spec.AccessModes = []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce}
	pv.Spec.PersistentVolumeReclaimPolicy = corev1.PersistentVolumeReclaimDelete
	pv.Spec.StorageClassName = "local-storage"
	pv.Spec.Local = &corev1.LocalVolumeSource{
		Path: path.Join(pm.dirManager.baseDir, latestStr),
	}
	pv.Spec.NodeAffinity = &corev1.VolumeNodeAffinity{
		Required: &corev1.NodeSelector{
			NodeSelectorTerms: []corev1.NodeSelectorTerm{
				{
					MatchExpressions: []corev1.NodeSelectorRequirement{
						{
							Key:      "kubernetes.io/hostname",
							Operator: corev1.NodeSelectorOpIn,
							Values:   []string{pm.hostname},
						},
					},
				},
			},
		},
	}

	_, err := pm.pvCli.Create(pv)
	return err
}
