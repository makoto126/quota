package main

import (
	log "github.com/sirupsen/logrus"
)

type quotaHandler struct {
}

func newQuotaHandler() *quotaHandler {
	return nil
}

func (qh *quotaHandler) OnAdd(obj interface{}) {

}

func (qh *quotaHandler) OnUpdate(oldObj, newObj interface{}) {
	log.Infoln("should quota")
}

func (qh *quotaHandler) OnDelete(obj interface{}) {
	log.Infoln("should dequota")
}
