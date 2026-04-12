package permissions

const (
	PostCreate PermissionBit = iota
	PostUpdate
	PostDelete
)

var PostPermCode = map[PermissionBit]string{
	PostCreate: "post:create",
	PostUpdate: "post:update",
	PostDelete: "post:delete",
}

var PostPermBit = map[string]PermissionBit{
	"post:create": PostCreate,
	"post:update": PostUpdate,
	"post:delete": PostDelete,
}

func init() {
	registerPerm(PostPermCode)
}