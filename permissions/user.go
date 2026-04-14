package permissions

var UserPermCode = map[PermissionBit]string{
	UserCreate: "user:create",
	UserUpdate: "user:update",
	UserDelete: "user:delete",
}

var UserPermBit = map[string]PermissionBit{
	"user:create": UserCreate,
	"user:update": UserUpdate,
	"user:delete": UserDelete,
}

func init() {
	registerPerm(UserPermCode)
}
