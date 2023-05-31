package tga

import (
	"testing"
)

func TestGetLastApkVersion(t *testing.T) {
	lastApkVersion, err := GetLastApkVersion()
	if err != nil {
		t.Fatalf("error in GetLastApkVersion")
	}
	if len(lastApkVersion) == 0 {
		t.Fatalf("expected non empty string from GetLastApkVersion")
	}
}
