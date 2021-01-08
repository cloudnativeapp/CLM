package utils

import "testing"

func TestVersionMatch(t *testing.T) {
	ok := VersionMatch("1.0.6", "1.0.1", "1.0.5")
	if ok {
		t.Log("ok")
	} else {
		t.Log("not ok")
	}
}
