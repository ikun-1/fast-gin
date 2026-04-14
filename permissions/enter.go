package permissions

// PermissionBit is the unified type for all permission bit identifiers.
type PermissionBit = uint

var PermCode = map[PermissionBit]string{}
var PermBit = map[string]PermissionBit{}

func registerPerm(permCode map[PermissionBit]string) {
	for bit, code := range permCode {
		PermCode[bit] = code
		PermBit[code] = bit
	}
}

const (
	// 用户权限
	UserCreate PermissionBit = iota
	UserUpdate
	UserDelete

	// 图片权限
	ImageUpload
	ImageDelete

	
)