package concept

import (
	"strconv"
	"strings"
)

const (
	_ROOT_PREFIX = "cassem/"
	_ELT_PREFIX  = _ROOT_PREFIX + "elements"
	_APP_PREFIX  = _ROOT_PREFIX + "apps"
	// _INS_PREFIX will be divided into two part, one is forward storage, another is reversed index.
	// 1. root/instances/normalized/instance-id => instance in detail
	// 2. root/instances/reversed/app-env-key => instances{instance-id}
	_INS_PREFIX        = _ROOT_PREFIX + "instances"
	_AGENT_PREFIX      = _ROOT_PREFIX + "agents"
	_VERSION_PREFIX    = "v"
	_ACL_POLICY_PREFIX = _ROOT_PREFIX + "acl/policy"
	_ACL_USER_PREFIX   = _ROOT_PREFIX + "acl/users"

	// utility constants, helps key to be more expressive.
	_SEP             = "/"
	_METADATA_SUFFIX = "/metadata"
)

func joint(keys ...string) string {
	return strings.Join(keys, _SEP)
}

// genElementKey generate element's key in storage, if any parameter is empty
// will touch off a panic.
func genElementKey(app, env, key string) string {
	if app == "" || env == "" || key == "" {
		panic("empty string could not be accepted")
	}
	return joint(_ELT_PREFIX, app, env, key)
}

func genAppKey(app string) string {
	return joint(_APP_PREFIX, app)
}

func genAppElementKey(app string) string {
	return joint(_ELT_PREFIX, app)
}

func genAppElementEnvKey(app, env string) string {
	return joint(_ELT_PREFIX, app, env)
}

func withVersion(key string, version int) string {
	if version < 1 {
		panic("invalid version: " + strconv.Itoa(version))
	}
	return joint(key, _VERSION_PREFIX+strconv.Itoa(version))
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
	return joint(_INS_PREFIX, "normalized", insId)
}

func genInstanceNormalDirKey() string {
	return joint(_INS_PREFIX, "normalized")
}

func genInstanceReversedKey(app, env, key string) string {
	k := app + "-" + env + "-" + key
	return joint(_INS_PREFIX, "reversed", k)
}

func genInstanceReversedKeyWithInsId(app, env, key string, insId string) string {
	k := app + "-" + env + "-" + key
	return joint(_INS_PREFIX, "reversed", k, insId)
}

func withAgentPrefix(agentId string) string {
	return joint(_AGENT_PREFIX, agentId)
}

func genAclPolicyKey() string {
	return _ACL_POLICY_PREFIX
}

func genUserKey(account string) string {
	if account == "" {
		return ""
	}

	return joint(_ACL_USER_PREFIX, account)
}
