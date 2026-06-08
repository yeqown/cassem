package concept

import (
	"strconv"
	"strings"
)

const (
	_ROOT_PREFIX = "cassem/"
	_ELT_PREFIX  = _ROOT_PREFIX + "elements"
	_APP_PREFIX  = _ROOT_PREFIX + "apps"
	_OP_PREFIX   = _ROOT_PREFIX + "operations"
	// _INS_PREFIX will be divided into two part, one is forward storage, another is reversed index.
	// 1. root/instances/normalized/instance-id => instance in detail
	// 2. root/instances/reversed/app-env-key => instances{instance-id}
	_INS_PREFIX        = _ROOT_PREFIX + "instances"
	_AGENT_PREFIX      = _ROOT_PREFIX + "agents"
	_VERSION_PREFIX    = "v"
	_ACL_POLICY_PREFIX = _ROOT_PREFIX + "acl/policy"
	_ACL_USER_PREFIX   = _ROOT_PREFIX + "acl/users"

	// utility constants, helps key to be more expressive.
	_SEP               = "/"
	_METADATA_SUFFIX   = "/metadata"
	_OPERATIONS_SUFFIX = "operations"
)

func joint(keys ...string) string {
	return strings.Join(keys, _SEP)
}

// genElementKey generate element's key in storage, if any parameter is empty
// will touch off a panic.
func GenElementKey(app, env, key string) string {
	if app == "" || env == "" || key == "" {
		panic("empty string could not be accepted")
	}
	return joint(_ELT_PREFIX, app, env, key)
}

func GenAppKey(app string) string {
	return joint(_APP_PREFIX, app)
}

func GenAppElementKey(app string) string {
	return joint(_ELT_PREFIX, app)
}

func GenAppElementEnvKey(app, env string) string {
	return joint(_ELT_PREFIX, app, env)
}

func WithVersion(key string, version int) string {
	if version < 1 {
		panic("invalid version: " + strconv.Itoa(version))
	}
	return joint(key, _VERSION_PREFIX+strconv.Itoa(version))
}

func WithMetadataSuffix(key string) string {
	return key + _METADATA_SUFFIX
}

func GenElementOperationDirKey(app, env, key string) string {
	return joint(_OP_PREFIX, app, env, key, _OPERATIONS_SUFFIX)
}

func GenElementOperationKey(app, env, key string, operatedAt int64) string {
	return joint(GenElementOperationDirKey(app, env, key), strconv.FormatInt(operatedAt, 10))
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
func ExtractPureKey(key string) string {
	arr := strings.Split(key, _SEP)
	if len(arr) <= 1 {
		return key
	}

	return arr[len(arr)-1]
}

func trimMetadata(key string) string {
	return strings.TrimSuffix(key, _METADATA_SUFFIX)
}

func GenInstanceNormalKey(insId string) string {
	return joint(_INS_PREFIX, "normalized", insId)
}

func GenInstanceNormalDirKey() string {
	return joint(_INS_PREFIX, "normalized")
}

func GenInstanceReversedKey(app, env, key string) string {
	k := app + "-" + env + "-" + key
	return joint(_INS_PREFIX, "reversed", k)
}

func GenInstanceReversedKeyWithInsId(app, env, key string, insId string) string {
	k := app + "-" + env + "-" + key
	return joint(_INS_PREFIX, "reversed", k, insId)
}

func WithAgentPrefix(agentId string) string {
	return joint(_AGENT_PREFIX, agentId)
}

func GenAclPolicyKey() string {
	return _ACL_POLICY_PREFIX
}

func GenUserKey(account string) string {
	if account == "" {
		return ""
	}

	return joint(_ACL_USER_PREFIX, account)
}
