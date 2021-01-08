package utils

import (
	"strconv"
	"strings"
)

const (
	ImplementNotFound         = "implement not found"
	ModuleStateAbnormal       = "module state abnormal"
	ReleaseStateAbnormal      = "release state abnormal"
	PreCheckWaiting           = "preCheck waiting"
	DependencyStateAbnormal   = "dependency state abnormal"
	DependencyVersionMismatch = "dependency version mismatch absent"
	DependencyAbsentError     = "dependency absent"
	DependencyPullError       = "dependency pull error"
	DependencyWaiting         = "dependency absent waiting"
)

const (
	Error int = iota - 2
	Warn
	Info
	Debug
)

func Contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

func Remove(list []string, s string) []string {
	for i, v := range list {
		if v == s {
			list = append(list[:i], list[i+1:]...)
		}
	}
	return list
}

func IgnoreWaitingErr(err error) error {
	if err == nil {
		return nil
	}
	e := err.Error()
	if e == ImplementNotFound ||
		e == PreCheckWaiting ||
		e == DependencyWaiting {
		return nil
	}
	return err
}

func IgnoreKnownErr(err error) error {
	if err == nil {
		return nil
	}
	e := err.Error()
	if e == ImplementNotFound ||
		e == ModuleStateAbnormal ||
		e == ReleaseStateAbnormal ||
		e == PreCheckWaiting ||
		e == DependencyStateAbnormal ||
		e == DependencyVersionMismatch ||
		e == DependencyAbsentError ||
		e == DependencyPullError ||
		e == DependencyWaiting {
		return nil
	}
	return err
}

//VersionMatch :  min <= current < max
func VersionMatch(current, min, max string) bool {
	if current == min || current == max {
		return true
	}
	if len(min) > 0 && !versionEqualOrNewer(current, min) {
		return false
	}
	if len(max) > 0 && versionEqualOrNewer(current, max) {
		return false
	}
	return true
}

func versionEqualOrNewer(v1, v2 string) bool {
	nslice := strings.Split(v1, ".")
	oslice := strings.Split(v2, ".")
	var maxlen int
	if len(nslice) > len(oslice) {
		maxlen = len(oslice)
	} else {
		maxlen = len(nslice)
	}

	for i := 0; i < maxlen; i++ {
		tmpn, err := strconv.Atoi(nslice[i])
		if err != nil {
			return true
		}

		tmpo, err := strconv.Atoi(oslice[i])
		if err != nil {
			return true
		}

		if tmpn > tmpo {
			return true
		}
	}

	return len(nslice) > len(oslice)
}
