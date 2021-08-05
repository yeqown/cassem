package concept

import (
	"strconv"
	"strings"
)

const (
	_ELT_PREFIX = "root/elements"
	_APP_PREFIX = "root/apps"
	// _INS_PREFIX will be divided into two part, one is forward storage, another is reversed index.
	// 1. root/instances/normalized/instance-id => instance in detail
	// 2. root/instances/reversed/app-env-key => instances{instance-id}
	_INS_PREFIX   = "root/instances"
	_AGENT_PREFIX = "root/agents"

	// utility constants, helps key to be more expressive.
	_SEP             = "/"
	_METADATA_SUFFIX = "/metadata"
)

// genElementKey generate element's key in storage, if any parameter is empty
// will touch off a panic.
func genElementKey(app, env, key string) string {
	if app == "" || env == "" || key == "" {
		panic("empty string could not be accepted")
	}

	return strings.Join([]string{_ELT_PREFIX, app, env, key}, _SEP)
}

func genAppKey(app string) string {
	return strings.Join([]string{_APP_PREFIX, app}, _SEP)
}

func genAppElementKey(app string) string {
	return strings.Join([]string{_ELT_PREFIX, app}, _SEP)
}

func genAppElementEnvKey(app, env string) string {
	return strings.Join([]string{_ELT_PREFIX, app, env}, _SEP)
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

// extractPureKey extract key=ele from sourceKey=root/element/app/env/ele
func extractPureKey(key string) string {
	arr := strings.Split(key, _SEP)
	if len(arr) <= 1 {
		return key
	}

	return arr[len(arr)-1]
}

func trimMetadata(key string) string {
	return strings.TrimSuffix(key, _METADATA_SUFFIX)
}

func genInstanceNormalKey(insId string) string {
	return strings.Join([]string{_INS_PREFIX, "normalized", insId}, _SEP)
}

func genInstanceReversedKey(app, env, key string) string {
	k := app + "-" + env + "-" + key
	return strings.Join([]string{_INS_PREFIX, "reversed", k}, _SEP)
}

func genInstanceReversedKeyWithInsid(app, env, key string, insId string) string {
	k := app + "-" + env + "-" + key
	return strings.Join([]string{_INS_PREFIX, "reversed", k, insId}, _SEP)
}
