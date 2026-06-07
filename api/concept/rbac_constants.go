package concept

// RBAC action constants
const (
	Action_READ    = "r"
	Action_WRITE   = "w"
	Action_DELETE  = "d"
	Action_PUBLISH = "p"
	Action_ANY     = "*"
)

// RBAC domain constants
const (
	Domain_ALL     = "*"
	Domain_CLUSTER = "cluster"
	// Domain_APP MUST NOT be used, this only represents the format of
	// app domain.
	Domain_APP = "app/env"
)

// RBAC role constants
const (
	// Role_SUPERADMIN can control whole resources.
	// p superadmin * * *
	Role_SUPERADMIN = "superadmin"
	// Role_ADMIN is an admin role who owns all apps' all permissions.
	Role_ADMIN = "admin"
	// Role_APPOWNER can only control the app's resources which belong to him
	// and visit other apps's resources.
	Role_APPOWNER = "appowner"
	// Role_DEVELOPER can only access(except delete, publish, rollback permissions)
	// the app's resources which belong to him and visit other apps's resources.
	Role_DEVELOPER = "appdeveloper"
	// Role_VISITOR can only access(readonly) app's resources.
	Role_VISITOR = "visitor"
)

// RBAC object constants
const (
	Object_USER    = "user"
	Object_ACL     = "acl"
	Object_APP     = "app"
	Object_ENV     = "env"
	Object_ELEMENT = "elem"
	Object_CLUSTER = "cluster"
	Object_ALL     = "*"
)
