package tga

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
)

var (
	kReScript = regexp.MustCompile(
		`AF_initDataCallback\({key:\s*'ds:5'.*?data:([\s\S]*?), sideChannel:.+<\/script`,
	)
)

func GetLastApkVersion() (string, error) {
	resp, err := http.Get("https://play.google.com/store/apps/details?id=com.app.tgtg&hl=en&gl=US")
	if err != nil {
		return "", fmt.Errorf("error from http.Get: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error from io.ReadAll: %w", err)
	}

	groups := kReScript.FindStringSubmatch(string(body))
	if len(groups) < 2 {
		return "", errors.New("unable to parse group in regular expression")
	}

	var parsed [][][]interface{}
	err = json.Unmarshal([]byte(groups[1]), &parsed)
	if err != nil {
		return "", fmt.Errorf("error from json.Unmarshal: %w", err)
	}

	subPart1, ok := parsed[1][2][140].([]interface{})
	if !ok {
		return "", fmt.Errorf("error from cast1: %w", err)
	}

	subPart2, ok := subPart1[0].([]interface{})
	if !ok {
		return "", fmt.Errorf("error from cast2: %w", err)
	}

	subPart3, ok := subPart2[0].([]interface{})
	if !ok {
		return "", fmt.Errorf("error from cast3: %w", err)
	}

	version, ok := subPart3[0].(string)
	if !ok {
		return "", fmt.Errorf("error from cast4: %w", err)
	}

	glog.Printf("parsed last apk version %v", version)

	return version, nil
}
