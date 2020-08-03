package main

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
)

//MntPoint is the data disk mount point
var MntPoint string

func setProjid(target string, prjid string) error {

	sub := fmt.Sprintf("project -s -p %s %s", target, prjid)
	return xfsQuota(sub)
}

func setQuota(quota string, prjid string) error {

	sub := fmt.Sprintf("limit -p bhard=%s %s", quota, prjid)
	return xfsQuota(sub)
}

func xfsQuota(sub string) error {

	cmd := exec.Command("xfs_quota", "-x", "-c", sub, MntPoint)
	stderr := new(bytes.Buffer)
	cmd.Stderr = stderr

	if err := cmd.Run(); err != nil {
		return err
	}
	if s := stderr.String(); s != "" {
		return errors.New(s)
	}
	return nil

}
