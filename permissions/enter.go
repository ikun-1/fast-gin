package permissions

// PermissionBit is the unified type for all permission bit identifiers.
type PermissionBit = uint

var PermCode = map[PermissionBit]string{}
var PermBit = map[string]PermissionBit{}

var index PermissionBit = 0

func registerPerm(permCode map[PermissionBit]string) {
	for _, code := range permCode {
		PermCode[index] = code
		PermBit[code] = index
		index++
	}
}