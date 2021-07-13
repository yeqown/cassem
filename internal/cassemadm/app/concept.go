package app

import (
	"strconv"
	"strings"
)

const (
	_ELT_PREFIX = "elements"
	_APP_PREFIX = "apps"
	_ENV_PREFIX = "envs"

	_SEP = "/"

	_METADATA_SUFFIX = "/metadata"
)

// genEltKey generate element's key in storage, if any parameter is empty
// will touch off a panic.
func genEltKey(app, env, eltKey string) string {
	if app == "" || env == "" || eltKey == "" {
		panic("empty string could not be accepted")
	}

	return strings.Join([]string{_ELT_PREFIX, app, env, eltKey}, _SEP)
}

func withVersion(key string, version int) string {
	if version < 1 {
		panic("invalid version: " + strconv.Itoa(version))
	}
	return key + "/v" + strconv.Itoa(version)
}

func withMetadataSuffix(key string) string {
	return key + _METADATA_SUFFIX
}

func trimVersion(key string) string {
	arr := strings.Split(key, _SEP)
	if len(arr) <= 1 {
		return key
	}
	// split result is not "vN" format
	if !strings.HasPrefix(arr[len(arr)-1], "v") {
		return key
	}

	return strings.Join(arr[:len(arr)-1], _SEP)
}

func trimMetadata(key string) string {
	return strings.TrimSuffix(key, _METADATA_SUFFIX)
}
