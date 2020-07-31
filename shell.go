package main

import (
	"fmt"
	"os/exec"
)

var mntPoint string

func setProjid(target string, prjid string) error {

	sub := fmt.Sprintf("'project -s -p %s %s'", target, prjid)
	return xfsQuota(sub)
}

func setQuota(quota string, prjid string) error {

	sub := fmt.Sprintf("'limit -p bhard=%s %s'", quota, prjid)
	return xfsQuota(sub)
}

func xfsQuota(sub string) error {

	cmd := exec.Command("xfs_quota", "-x", "-c", sub, mntPoint)
	return cmd.Run()
}
