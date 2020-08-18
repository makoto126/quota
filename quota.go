package main

import (
	"encoding/json"
	"strings"

	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/types"
	typev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

const (
	annoKey = "quota"
)

type quotaHandler struct {
	corev1Cli typev1.CoreV1Interface
}

func (qh *quotaHandler) OnAdd(obj interface{}) {
	//nothing to do
}

func (qh *quotaHandler) OnDelete(obj interface{}) {
	//nothing to do
}

func (qh *quotaHandler) OnUpdate(oldObj, newObj interface{}) {

	newPvc := newObj.(*corev1.PersistentVolumeClaim)

	if newPvc.Spec.StorageClassName == nil || *newPvc.Spec.StorageClassName != StorageClassName {
		return
	}
	if newPvc.Status.Phase != corev1.ClaimBound {
		return
	}
	if !strings.HasPrefix(newPvc.Spec.VolumeName, NodeName) {
		return
	}

	oldPvc := oldObj.(*corev1.PersistentVolumeClaim)

	firstBound := oldPvc.Status.Phase != corev1.ClaimBound
	if firstBound {
		storage := newPvc.Spec.Resources.Requests["storage"]

		patchJSON := map[string]interface{}{
			"metadata": map[string]interface{}{
				"annotations": map[string]interface{}{
					annoKey: storage.String(),
				},
			},
		}

		patchData, err := json.Marshal(patchJSON)
		if err != nil {
			log.Errorln(err)
			return
		}

		pvcCli := qh.corev1Cli.PersistentVolumeClaims(newPvc.Namespace)
		_, err = pvcCli.Patch(newPvc.Name, types.StrategicMergePatchType, patchData)
		if err != nil {
			log.Errorln(err)
		}
		return
	}

	volumeChanged := oldPvc.Spec.VolumeName != newPvc.Spec.VolumeName
	quotaChanged := oldPvc.Annotations[annoKey] != newPvc.Annotations[annoKey]
	if volumeChanged || quotaChanged {

		expected := convertStorageUnit(newPvc.Annotations[annoKey])
		eq, err := resource.ParseQuantity(expected)
		if err != nil {
			log.Errorln(err)
			return
		}

		projid := getProjidFromVolumeName(newPvc.Spec.VolumeName)
		used, _, err := getUsedQuota(projid)
		if err != nil {
			log.Errorln(err)
			return
		}
		uq, err := resource.ParseQuantity(used)

		if eq.Cmp(uq) == -1 {
			//should alert
			log.Errorf("pvc %s quota is lower than used", newPvc.Name)
			return
		}

		if err := setQuota(expected, projid); err != nil {
			log.Errorln(err)
		}
	}
}
