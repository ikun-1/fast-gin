package compare

import "reflect"

func Equal(x, y any) bool {
	// 特殊处理 slice 和 map 的 nil 情况
	xv, yv := reflect.ValueOf(x), reflect.ValueOf(y)
	if (xv.Kind() == reflect.Slice || xv.Kind() == reflect.Map) &&
		xv.Kind() == yv.Kind() {
		if xv.IsNil() || yv.IsNil() {
			return xv.Len() == yv.Len()
		}
	}

	// 其他情况直接用 reflect.DeepEqual
	return reflect.DeepEqual(x, y)
}
