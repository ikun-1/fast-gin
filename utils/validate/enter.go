package validate

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/locales/zh"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	zh_translations "github.com/go-playground/validator/v10/translations/zh"
	"go.uber.org/zap"
)

type validateRule func(fl validator.FieldLevel) bool
type ValidateConfig struct {
	Tag         string
	Validate    validateRule
	Translation string
	Override    bool
}

var (
	validations = make(map[string]ValidateConfig)
	validate    *validator.Validate
	translator  ut.Translator
	once        sync.Once
)

// InitValidator 初始化验证器和翻译器（方法一 + 自定义）
func InitValidator() {
	once.Do(func() {
		// 1. 创建中文翻译器
		zhLocale := zh.New()
		uni := ut.New(zhLocale, zhLocale)
		translator, _ = uni.GetTranslator("zh")
		// 2. 获取 Gin 框架中的验证器实例
		if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
			validate = v
		}

		// 3. 注册官方默认中文翻译
		if err := zh_translations.RegisterDefaultTranslations(validate, translator); err != nil {
			panic(err)
		}

		// 4. 注册自定义验证规则
		for _, st := range validations {
			st.registerValidation()
		}

		zap.L().Info("验证器和翻译器初始化成功")
	})
}

// GetValidationErrorMessages 获取验证错误消息（支持自定义翻译）
func GetValidationErrorMessages(err error, model any) map[string]string {
	if err == nil {
		return nil
	}

	messageMap := make(map[string]string)

	switch e := err.(type) {
	case *json.SyntaxError:
		messageMap["_error"] = "JSON 格式错误"
		return messageMap

	case *json.UnmarshalTypeError:
		fieldName := getJsonFieldName(e.Field, model)
		messageMap[fieldName] = fmt.Sprintf("字段应该是%s类型", e.Type.String())
		return messageMap

	case validator.ValidationErrors:
		for _, validationErr := range e {
			fieldName := validationErr.Field()
			jsonFieldName := getJsonFieldName(fieldName, model)

			// 优先使用结构体标签中的自定义消息
			var msg string
			if model != nil {
				if fieldType, found := getFieldTypeByName(model, fieldName); found {
					if customMsg := fieldType.Tag.Get("msg"); customMsg != "" {
						messageMap[jsonFieldName] = customMsg
						continue
					}
				}
			}

			// 使用翻译器获取消息（这里会用到我们的自定义翻译）
			msg = validationErr.Translate(translator)

			displayName := getFieldDisplayName(fieldName, model)
			msg = strings.Replace(msg, fieldName, displayName, 1)
			if tag, param := validationErr.Tag(), validationErr.Param(); strings.Contains(strings.ToLower(tag), "field") {
				displayName := getFieldDisplayName(param, model)
				msg = strings.Replace(msg, param, displayName, 1)
			}

			messageMap[jsonFieldName] = msg
		}
		return messageMap

	default:
		messageMap["_error"] = "请求参数错误"
		return messageMap
	}
}

func getJsonFieldName(fieldName string, model any) string {
	name := strings.ToLower(fieldName)
	if model == nil {
		return name
	}
	fieldType, found := getFieldTypeByName(model, fieldName)
	if !found {
		return name
	}
	// 使用 json 标签
	if jsonTag := fieldType.Tag.Get("json"); jsonTag != "" {
		parts := strings.Split(jsonTag, ",")
		if parts[0] != "" && parts[0] != "-" {
			return parts[0]
		}
	}
	return name
}

// getFieldDisplayName 获取字段显示名（辅助函数）
func getFieldDisplayName(fieldName string, model any) string {
	name := strings.ToLower(fieldName)
	if model == nil {
		return name
	}

	fieldType, found := getFieldTypeByName(model, fieldName)
	if !found {
		return name
	}

	// 优先使用 display 标签
	if displayName := fieldType.Tag.Get("display"); displayName != "" {
		return displayName
	}

	// 其次使用 label 标签
	if label := fieldType.Tag.Get("label"); label != "" {
		return label
	}

	// 使用 json 标签
	if jsonTag := fieldType.Tag.Get("json"); jsonTag != "" {
		parts := strings.Split(jsonTag, ",")
		if parts[0] != "" && parts[0] != "-" {
			return parts[0]
		}
	}
	return name
}

// getFieldTypeByName 获取字段类型（辅助函数）
func getFieldTypeByName(model any, name string) (reflect.StructField, bool) {
	if model == nil {
		return reflect.StructField{}, false
	}

	typ := reflect.TypeOf(model)
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	return typ.FieldByName(name)
}

// RegisterValidation 注册自定义验证规则
func (vc *ValidateConfig) registerValidation() {
	validate.RegisterValidation(vc.Tag, func(fl validator.FieldLevel) bool {
		return vc.Validate(fl)
	})

	validate.RegisterTranslation(vc.Tag, translator,
		func(ut ut.Translator) error {
			return ut.Add(vc.Tag, vc.Translation, vc.Override)
		},
		func(ut ut.Translator, fe validator.FieldError) string {
			t, _ := ut.T(fe.Tag(), fe.Field(), fe.Param())
			return t
		},
	)
}

func ValidateError(err error) string {
	errs, ok := err.(validator.ValidationErrors)
	if !ok {
		return err.Error()
	}
	var list []string
	for _, e := range errs {
		list = append(list, e.Translate(translator))
	}
	return strings.Join(list, ";")
}
