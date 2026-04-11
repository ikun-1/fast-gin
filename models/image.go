package models

type ImageModel struct {
	Model
	Address  string `gorm:"size:255" json:"address"`  // 图片地址
	FileName string `gorm:"size:255" json:"fileName"` // 原始文件名
	FileHash string `gorm:"size:64" json:"fileHash"`  // 文件MD5哈希
	UserID   uint   `json:"userID"`                   // 上传用户ID
}
