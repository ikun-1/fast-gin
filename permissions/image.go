package permissions

const (
	ImageUpload PermissionBit = iota
	ImageDelete
)

var ImagePermCode = map[PermissionBit]string{
	ImageUpload: "image:upload",
	ImageDelete: "image:delete",
}

var ImagePermBit = map[string]PermissionBit{
	"image:upload": ImageUpload,
	"image:delete": ImageDelete,
}

func init() {
	registerPerm(ImagePermCode)
}