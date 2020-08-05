package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
)

func setProjid(target string, prjid string) error {

	sub := fmt.Sprintf("project -s -p %s %s", target, prjid)
	return xfsQuota(sub, nil)
}

func setQuota(quota string, prjid string) error {

	sub := fmt.Sprintf("limit -p bhard=%s %s", quota, prjid)
	return xfsQuota(sub, nil)
}

type report struct {
	Projid string
	Used   uint64
	Quota  uint64
}

func getReport() ([]*report, error) {

	sub := "report"
	stdout := new(bytes.Buffer)
	if err := xfsQuota(sub, stdout); err != nil {
		return nil, err
	}

	res := []*report{}

	var line string
	var err error
	for {
		line, err = stdout.ReadString('\n')
		if err != nil {
			break
		}
		if strings.HasPrefix(line, "#") {
			lf := strings.Fields(line)

			projid := lf[0]
			if projid == "#0" {
				continue
			}

			used, err := strconv.ParseUint(lf[1], 10, 64)
			if err != nil {
				return nil, err
			}
			quota, err := strconv.ParseUint(lf[3], 10, 64)
			if err != nil {
				return nil, err
			}
			res = append(res, &report{
				Projid: projid,
				Used:   used,
				Quota:  quota,
			})
		}
	}

	return res, nil
}

func xfsQuota(sub string, stdout io.Writer) error {

	cmd := exec.Command("xfs_quota", "-x", "-c", sub, BaseDir)

	cmd.Stdout = stdout

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
