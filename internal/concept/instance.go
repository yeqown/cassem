package concept

// InstanceID returns the stable identity used for registration and delivery.
func InstanceID(instance *Instance) string {
	if instance.GetClientId() == "" {
		return "cassem@" + instance.GetClientIp()
	}
	return instance.GetClientId() + "@" + instance.GetClientIp()
}
