package models

type Image struct {
	Model
	Address  string `gorm:"size:255;comment:图片地址" json:"address"`    // 图片地址
	FileName string `gorm:"size:255;comment:原始文件名" json:"fileName"`  // 原始文件名
	FileHash string `gorm:"size:64;comment:文件MD5哈希" json:"fileHash"` // 文件MD5哈希
	UserID   uint   `gorm:"comment:上传用户ID" json:"userID"`            // 上传用户ID
}

func init() {
	MigrateModels = append(MigrateModels, &Image{})
}
