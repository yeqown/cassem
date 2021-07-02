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
