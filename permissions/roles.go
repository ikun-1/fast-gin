package permissions

const (
	RoleAdmin int8 = iota
	RoleUser
)

var RoleCode = map[int8]string{
	RoleAdmin: "admin",
	RoleUser:  "user",
}

var Role = map[string]int8{
	"admin": RoleAdmin,
	"user":  RoleUser,
}