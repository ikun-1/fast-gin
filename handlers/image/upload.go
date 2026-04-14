package image

import (
	"errors"
	"fast-gin/dal/query"
	"fast-gin/global"
	"fast-gin/middleware"
	"fast-gin/models"
	"fast-gin/utils/res"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"fast-gin/utils/find"
	"fast-gin/utils/md5"
	"fast-gin/utils/random"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

var whiteList = []string{
	".jpg",
	".jpeg",
	".png",
	".webp",
}

// UploadView 上传图片
// @Summary      上传图片
// @Description  上传图片文件，支持jpg、jpeg、png、webp格式，上传成功后返回图片ID和访问地址
// @Tags         image
// @Accept       multipart/form-data
// @Produce      json
// @Security     Bearer
// @Param        file  formData  file  true   "图片文件（支持jpg、jpeg、png、webp，最大2MB）"
// @Success      200   {object}  res.Response  "{"code":0,"msg":"上传成功","data":{"id":1,"address":"/uploads/images/xxx.jpg"}}"
// @Failure      200   {object}  res.Response       "{"code":1,"msg":"请选择文件"}"
// @Failure      200   {object}  res.Response       "{"code":3,"msg":"用户认证失败"}"
// @Router       /images [post]
func (Image) UploadView(c *gin.Context) {
	// 获取当前用户ID
	claims := middleware.GetAuth(c)
	if claims == nil || claims.UserID == 0 {
		res.FailWithMsg(c, "用户认证失败")
		return
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		res.FailWithMsg(c, "请选择文件")
		return
	}

	// 大小限制
	if fileHeader.Size > global.Config.Upload.Size*1024*1024 {
		res.FailWithMsg(c, "上传文件过大")
		return
	}
	// 后缀判断
	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))

	if !find.InList(whiteList, ext) {
		res.FailWithMsg(c, "上传文件后缀非法")
		return
	}

	// 处理文件名重复
	// uploads/images/xx.jpg
	// uploads/images/xx_7hf.jpg
	fp := path.Join("uploads", global.Config.Upload.Dir, fileHeader.Filename)
	var fileHash string
	for {
		_, err1 := os.Stat(fp)
		if os.IsNotExist(err1) {
			break
		}
		// 文件存在
		// 算上传的图片和本身的图片是不是一样的，如果是一样的，那就直接返回之前的地址
		uploadFile, _ := fileHeader.Open()
		oldFile, _ := os.Open(fp)

		uploadFileHash := md5.MD5WithFile(uploadFile)
		oldFileHash := md5.MD5WithFile(oldFile)
		if uploadFileHash == oldFileHash {
			// 上传的图片，名称和内容都是一样的，检查数据库中是否已有记录
			fileHash = uploadFileHash
			imageModel, dbErr := query.Image.WithContext(c).Where(query.Image.FileHash.Eq(fileHash)).Take()
			if dbErr == nil {
				// 数据库中已有记录，直接返回
				res.Ok(c, gin.H{
					"id":      imageModel.ID,
					"address": imageModel.Address,
				}, "上传成功")
				return
			}
			if !errors.Is(dbErr, gorm.ErrRecordNotFound) {
				res.FailWithMsg(c, "查询图片记录失败")
				return
			}
			// 数据库中没有记录，创建新记录
			imageModel = &models.Image{
				Address:  "/" + fp,
				FileName: fileHeader.Filename,
				FileHash: fileHash,
				UserID:   claims.UserID,
			}
			if dbErr = query.Image.WithContext(c).Create(imageModel); dbErr != nil {
				res.FailWithMsg(c, "图片信息保存失败")
				return
			}
			res.Ok(c, gin.H{
				"id":      imageModel.ID,
				"address": imageModel.Address,
			}, "上传成功")
			return
		}
		// 上传的图片，名称是一样的，但是内容不一样
		fileNameNotExt := strings.TrimSuffix(fileHeader.Filename, ext)
		newFileName := fmt.Sprintf("%s_%s%s", fileNameNotExt, random.RandStr(3), ext)
		fp = path.Join("uploads", global.Config.Upload.Dir, newFileName)
	}
	c.SaveUploadedFile(fileHeader, fp)

	// 计算上传文件的hash
	uploadFile, _ := fileHeader.Open()
	fileHash = md5.MD5WithFile(uploadFile)

	// 保存图片信息到数据库
	imageModel := &models.Image{
		Address:  "/" + fp,
		FileName: fileHeader.Filename,
		FileHash: fileHash,
		UserID:   claims.UserID,
	}
	if err = query.Image.WithContext(c).Create(imageModel); err != nil {
		res.FailWithMsg(c, "图片信息保存失败")
		return
	}

	res.Ok(c, gin.H{
		"id":      imageModel.ID,
		"address": imageModel.Address,
	}, "上传成功")
}
