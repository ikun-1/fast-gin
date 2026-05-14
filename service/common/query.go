package common

import (
	"fast-gin/global"
	"fast-gin/models"
	"fmt"
	"regexp"
	"strings"

	"gorm.io/gen/field"
	"gorm.io/gorm"
)

var sortFieldRegexp = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

// QueryOption 查询选项，包含分页、模糊查询、自定义条件、预加载、调试等
type QueryOption struct {
	models.PageInfo             // 包含分页、搜索、排序参数
	LikeFields  []field.String // 类型安全的模糊查询字段（传 query.User.Username 等）
	Where       *gorm.DB       // 自定义 Where 条件（可选）
	Preloads    []string       // 预加载的关联字段
	Debug       bool           // 是否开启调试模式
}

func buildSafeOrder(option QueryOption) string {
	dir := strings.ToLower(option.SortDir)
	if dir != "asc" && dir != "desc" {
		dir = "desc"
	}

	field := strings.TrimSpace(option.SortBy)
	if field == "" || !sortFieldRegexp.MatchString(field) {
		field = "created_at"
	}

	return fmt.Sprintf("%s %s", field, strings.ToUpper(dir))
}

// QueryList 通用列表查询函数
// 参数：
//   model: 要查询的模型实例（用于指定表和基础条件）
//   option: 查询选项
// 返回：
//   list: 查询结果列表
//   count: 符合条件的总条数（不分页）
//   err: 查询过程中的错误
func QueryList[T any](model T, option QueryOption) (list []T, count int64, err error) {
	list = make([]T, 0)

	// 1. 初始化查询链，基于全局 DB 实例，绑定基础模型条件
	query := global.DB.Model(&model)

	// 2. 开启调试模式（如果需要）
	if option.Debug {
		query = query.Debug()
	}

	// 3. 基础条件：model 本身的字段筛选（如传入非空字段作为等值条件）
	query = query.Where(&model)

	// 4. 自定义 Where 条件（如果有）
	if option.Where != nil {
		query = query.Where(option.Where)
	}

	// 5. 模糊查询（Key 不为空且有指定模糊字段时）
	if option.Key != "" && len(option.LikeFields) > 0 {
		likeSQL := strings.Builder{}
		likeArgs := make([]interface{}, 0, len(option.LikeFields))
		likeSQL.WriteString("(")
		for i, f := range option.LikeFields {
			if i > 0 {
				likeSQL.WriteString(" OR ")
			}
			likeSQL.WriteString(string(f.ColumnName()) + " LIKE ?")
			likeArgs = append(likeArgs, "%"+option.Key+"%")
		}
		likeSQL.WriteString(")")
		query = query.Where(likeSQL.String(), likeArgs...)
	}

	// 6. 预加载关联字段
	for _, preload := range option.Preloads {
		query = query.Preload(preload)
	}

	// 7. 处理总条数（分页时需要先查总数，注意：Count 会忽略 Limit/Offset/Order）
	if err = query.Count(&count).Error; err != nil {
		return nil, 0, fmt.Errorf("查询总数失败: %w", err)
	}

	// 8. 分页参数处理
	if option.Page <= 0 {
		option.Page = 1 // 页码默认从 1 开始
	}
	if option.Limit <= 0 {
		option.Limit = -1 // Limit(-1) 表示取消限制，返回所有数据
	}
	offset := (option.Page - 1) * option.Limit

	// 9. 排序（单字段字符串）
	query = query.Order(buildSafeOrder(option))

	// 10. 执行分页查询
	if option.Limit != -1 {
		query = query.Limit(option.Limit).Offset(offset)
	}
	if err = query.Find(&list).Error; err != nil {
		return nil, 0, fmt.Errorf("查询列表失败: %w", err)
	}

	return list, count, nil
}
