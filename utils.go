package main

import "strings"

func getProjidFromVolumeName(name string) string {
	sp := strings.Split(name, "-")
	return sp[len(sp)-1]
}

//convert Mi Gi Ti to M G T
func convertStorageUnit(val string) string {
	units := []string{"Mi", "Gi", "Ti"}

	for _, u := range units {
		if strings.HasSuffix(val, u) {
			return val[:len(val)-1]
		}
	}

	return val
}
