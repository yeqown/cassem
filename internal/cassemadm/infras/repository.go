package infras

// Repository describes all methods should storage component should support.
type Repository interface {
	AddUser()
	EditUser()

	RBAC()
}
