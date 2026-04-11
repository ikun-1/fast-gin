package common

import (
	"fast-gin/global"
	"fast-gin/models"
	"fmt"
	"gorm.io/gorm"
)

// QueryOption 查询选项，包含分页、模糊查询、自定义条件、预加载、调试等
type QueryOption struct {
	models.PageInfo          // 包含 Page(int), Limit(int), Key(string), Order(string)
	Likes           []string // 模糊查询的字段列表
	Where           *gorm.DB // 自定义 Where 条件（可选）
	Preloads        []string // 预加载的关联字段
	Debug           bool     // 是否开启调试模式
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
	if option.Key != "" && len(option.Likes) > 0 {
		// 构建模糊查询条件：OR 连接多个字段的 LIKE
		likeExpr := ""
		likeArgs := make([]interface{}, 0, len(option.Likes))
		for i, column := range option.Likes {
			if i > 0 {
				likeExpr += " OR "
			}
			likeExpr += fmt.Sprintf("%s LIKE ?", column)
			likeArgs = append(likeArgs, fmt.Sprintf("%%%s%%", option.Key))
		}
		query = query.Where(likeExpr, likeArgs...)
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

	// 9. 排序（默认按创建时间降序）
	if option.Order == "" {
		option.Order = "created_at desc"
	}
	query = query.Order(option.Order)

	// 10. 执行分页查询
	if option.Limit != -1 {
		query = query.Limit(option.Limit).Offset(offset)
	}
	if err = query.Find(&list).Error; err != nil {
		return nil, 0, fmt.Errorf("查询列表失败: %w", err)
	}

	return list, count, nil
}
