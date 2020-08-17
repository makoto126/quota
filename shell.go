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

func setProjid(target string, projid string) error {

	sub := fmt.Sprintf("project -s -p %s %s", target, projid)
	return xfsQuota(sub, nil)
}

func setQuota(quota string, projid string) error {

	sub := fmt.Sprintf("limit -p bhard=%s %s", quota, projid)
	return xfsQuota(sub, nil)
}

func getUsedQuota(projid string) (string, string, error) {

	sub := "report -h"
	stdout := new(bytes.Buffer)
	if err := xfsQuota(sub, stdout); err != nil {
		return "", "", err
	}

	var line string
	var err error
	for {
		line, err = stdout.ReadString('\n')
		if err != nil {
			break
		}
		prefix := "#" + projid
		if strings.HasPrefix(line, prefix) {
			lf := strings.Fields(line)
			return lf[1], lf[3], nil
		}
	}

	return "", "", fmt.Errorf("projid %s not found", projid)
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

			//remove #
			projid := lf[0][1:]
			if projid == "0" {
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
