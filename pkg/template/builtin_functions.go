// Package template 提供模板处理功能，支持模板渲染和缓存
package template

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html"
	"math"
	"math/rand"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"
)

// registerBuiltinFunctions 注册所有内置函数
func (e *Engine) registerBuiltinFunctions() {
	// 字符串操作函数
	e.registerStringFunctions()

	// 日期与时间函数
	e.registerDateTimeFunctions()

	// 数学运算函数
	e.registerMathFunctions()

	// 数据转换函数
	e.registerConversionFunctions()

	// 集合操作函数
	e.registerCollectionFunctions()

	// 条件逻辑函数
	e.registerConditionalFunctions()

	// 加密与编码函数
	e.registerCryptoFunctions()
}

// registerStringFunctions 注册字符串操作函数
func (e *Engine) registerStringFunctions() {
	// 字符串大小写转换
	e.funcs["toUpper"] = strings.ToUpper
	e.funcs["toLower"] = strings.ToLower
	e.funcs["title"] = strings.Title

	// 字符串修剪
	e.funcs["trim"] = strings.TrimSpace
	e.funcs["trimPrefix"] = strings.TrimPrefix
	e.funcs["trimSuffix"] = strings.TrimSuffix

	// 字符串替换
	e.funcs["replace"] = strings.Replace
	e.funcs["replaceAll"] = strings.ReplaceAll

	// 字符串分割与连接
	e.funcs["split"] = strings.Split
	e.funcs["join"] = strings.Join

	// 字符串包含检查
	e.funcs["contains"] = strings.Contains
	e.funcs["hasPrefix"] = strings.HasPrefix
	e.funcs["hasSuffix"] = strings.HasSuffix

	// 字符串长度
	e.funcs["length"] = func(s string) int {
		return len(s)
	}

	// 正则表达式
	e.funcs["regexMatch"] = func(pattern, s string) bool {
		match, _ := regexp.MatchString(pattern, s)
		return match
	}

	e.funcs["regexReplace"] = func(pattern, repl, s string) string {
		re, err := regexp.Compile(pattern)
		if err != nil {
			return s
		}
		return re.ReplaceAllString(s, repl)
	}

	// URL编码/解码
	e.funcs["urlEncode"] = func(s string) string {
		return url.QueryEscape(s)
	}
	e.funcs["urlDecode"] = func(s string) string {
		result, _ := url.QueryUnescape(s)
		return result
	}

	// HTML转义
	e.funcs["htmlEscape"] = html.EscapeString
	e.funcs["htmlUnescape"] = html.UnescapeString

	// 子字符串
	e.funcs["substr"] = func(s string, start, length int) string {
		if start < 0 {
			start = 0
		}
		if start > len(s) {
			return ""
		}
		end := start + length
		if end > len(s) {
			end = len(s)
		}
		return s[start:end]
	}

	// 重复字符串
	e.funcs["repeat"] = strings.Repeat
}

// registerDateTimeFunctions 注册日期时间函数
func (e *Engine) registerDateTimeFunctions() {
	// 当前时间
	e.funcs["now"] = time.Now

	// 格式化时间
	e.funcs["formatTime"] = func(t time.Time, layout string) string {
		return t.Format(layout)
	}

	// 常用日期格式
	e.funcs["formatDate"] = func(t time.Time) string {
		return t.Format("2006-01-02")
	}

	e.funcs["formatDateTime"] = func(t time.Time) string {
		return t.Format("2006-01-02 15:04:05")
	}

	// 时间解析
	e.funcs["parseTime"] = func(layout, value string) (time.Time, error) {
		return time.Parse(layout, value)
	}

	// 时间操作
	e.funcs["addDate"] = func(t time.Time, days int) time.Time {
		return t.AddDate(0, 0, days)
	}

	e.funcs["addHours"] = func(t time.Time, hours int) time.Time {
		return t.Add(time.Duration(hours) * time.Hour)
	}

	e.funcs["addMinutes"] = func(t time.Time, minutes int) time.Time {
		return t.Add(time.Duration(minutes) * time.Minute)
	}

	// 时间差
	e.funcs["since"] = time.Since
	e.funcs["until"] = time.Until

	// 时间比较
	e.funcs["isAfter"] = func(a, b time.Time) bool {
		return a.After(b)
	}

	e.funcs["isBefore"] = func(a, b time.Time) bool {
		return a.Before(b)
	}

	// 获取时间组件
	e.funcs["year"] = func(t time.Time) int {
		return t.Year()
	}

	e.funcs["month"] = func(t time.Time) string {
		return t.Month().String()
	}

	e.funcs["day"] = func(t time.Time) int {
		return t.Day()
	}

	e.funcs["weekday"] = func(t time.Time) string {
		return t.Weekday().String()
	}

	// 时间戳
	e.funcs["unixTime"] = func(t time.Time) int64 {
		return t.Unix()
	}

	e.funcs["fromUnixTime"] = func(sec int64) time.Time {
		return time.Unix(sec, 0)
	}
}

// registerMathFunctions 注册数学运算函数
func (e *Engine) registerMathFunctions() {
	// 基本运算
	e.funcs["add"] = func(a, b float64) float64 {
		return a + b
	}

	e.funcs["sub"] = func(a, b float64) float64 {
		return a - b
	}

	e.funcs["mul"] = func(a, b float64) float64 {
		return a * b
	}

	e.funcs["div"] = func(a, b float64) float64 {
		if b == 0 {
			return 0
		}
		return a / b
	}

	e.funcs["mod"] = math.Mod

	// 取整
	e.funcs["ceil"] = math.Ceil
	e.funcs["floor"] = math.Floor
	e.funcs["round"] = math.Round

	// 最大最小值
	e.funcs["max"] = math.Max
	e.funcs["min"] = math.Min

	// 绝对值
	e.funcs["abs"] = math.Abs

	// 幂运算
	e.funcs["pow"] = math.Pow
	e.funcs["sqrt"] = math.Sqrt

	// 随机数
	e.funcs["rand"] = func() float64 {
		return rand.Float64()
	}

	e.funcs["randInt"] = func(min, max int) int {
		return rand.Intn(max-min) + min
	}
}

// registerConversionFunctions 注册数据转换函数
func (e *Engine) registerConversionFunctions() {
	// 类型转换
	e.funcs["toString"] = func(v interface{}) string {
		return fmt.Sprintf("%v", v)
	}

	e.funcs["toInt"] = func(v interface{}) int {
		switch val := v.(type) {
		case int:
			return val
		case int64:
			return int(val)
		case float64:
			return int(val)
		case string:
			var i int
			fmt.Sscanf(val, "%d", &i)
			return i
		default:
			return 0
		}
	}

	e.funcs["toFloat"] = func(v interface{}) float64 {
		switch val := v.(type) {
		case float64:
			return val
		case int:
			return float64(val)
		case int64:
			return float64(val)
		case string:
			var f float64
			fmt.Sscanf(val, "%f", &f)
			return f
		default:
			return 0
		}
	}

	e.funcs["toBool"] = func(v interface{}) bool {
		switch val := v.(type) {
		case bool:
			return val
		case int:
			return val != 0
		case string:
			return val != "" && val != "0" && val != "false"
		default:
			return false
		}
	}

	// JSON操作
	e.funcs["jsonEncode"] = func(v interface{}) string {
		bytes, err := json.Marshal(v)
		if err != nil {
			return "{}"
		}
		return string(bytes)
	}

	e.funcs["jsonDecode"] = func(s string) interface{} {
		var data interface{}
		err := json.Unmarshal([]byte(s), &data)
		if err != nil {
			return nil
		}
		return data
	}

	e.funcs["prettifyJSON"] = func(s string) string {
		var data interface{}
		err := json.Unmarshal([]byte(s), &data)
		if err != nil {
			return s
		}
		pretty, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			return s
		}
		return string(pretty)
	}
}

// registerCollectionFunctions 注册集合操作函数
func (e *Engine) registerCollectionFunctions() {
	// 数组/切片操作
	e.funcs["first"] = func(a []interface{}) interface{} {
		if len(a) == 0 {
			return nil
		}
		return a[0]
	}

	e.funcs["last"] = func(a []interface{}) interface{} {
		if len(a) == 0 {
			return nil
		}
		return a[len(a)-1]
	}

	e.funcs["slice"] = func(a []interface{}, start, end int) []interface{} {
		if start < 0 {
			start = 0
		}
		if end > len(a) {
			end = len(a)
		}
		return a[start:end]
	}

	e.funcs["append"] = func(a []interface{}, v interface{}) []interface{} {
		return append(a, v)
	}

	e.funcs["indexOf"] = func(a []interface{}, v interface{}) int {
		for i, item := range a {
			if item == v {
				return i
			}
		}
		return -1
	}

	e.funcs["reverse"] = func(a []interface{}) []interface{} {
		reversed := make([]interface{}, len(a))
		for i, item := range a {
			reversed[len(a)-i-1] = item
		}
		return reversed
	}

	// Map操作
	e.funcs["keys"] = func(m map[string]interface{}) []string {
		keys := make([]string, 0, len(m))
		for k := range m {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		return keys
	}

	e.funcs["values"] = func(m map[string]interface{}) []interface{} {
		values := make([]interface{}, 0, len(m))
		for _, v := range m {
			values = append(values, v)
		}
		return values
	}

	e.funcs["hasKey"] = func(m map[string]interface{}, key string) bool {
		_, ok := m[key]
		return ok
	}

	// 集合聚合
	e.funcs["sum"] = func(a []float64) float64 {
		sum := 0.0
		for _, v := range a {
			sum += v
		}
		return sum
	}

	e.funcs["avg"] = func(a []float64) float64 {
		if len(a) == 0 {
			return 0
		}
		return e.funcs["sum"].(func([]float64) float64)(a) / float64(len(a))
	}
}

// registerConditionalFunctions 注册条件逻辑函数
func (e *Engine) registerConditionalFunctions() {
	// 条件选择
	e.funcs["ternary"] = func(condition bool, trueVal, falseVal interface{}) interface{} {
		if condition {
			return trueVal
		}
		return falseVal
	}

	e.funcs["defaultValue"] = func(val, defaultVal interface{}) interface{} {
		if val == nil {
			return defaultVal
		}
		return val
	}

	e.funcs["coalesce"] = func(values ...interface{}) interface{} {
		for _, v := range values {
			if v != nil {
				// 检查空字符串
				if s, ok := v.(string); ok && s == "" {
					continue
				}
				return v
			}
		}
		return nil
	}

	// 逻辑操作
	e.funcs["and"] = func(a, b bool) bool {
		return a && b
	}

	e.funcs["or"] = func(a, b bool) bool {
		return a || b
	}

	e.funcs["not"] = func(a bool) bool {
		return !a
	}

	// 比较
	e.funcs["eq"] = func(a, b interface{}) bool {
		return a == b
	}

	e.funcs["ne"] = func(a, b interface{}) bool {
		return a != b
	}

	e.funcs["lt"] = func(a, b float64) bool {
		return a < b
	}

	e.funcs["le"] = func(a, b float64) bool {
		return a <= b
	}

	e.funcs["gt"] = func(a, b float64) bool {
		return a > b
	}

	e.funcs["ge"] = func(a, b float64) bool {
		return a >= b
	}

	// 字符串比较
	e.funcs["strEq"] = func(a, b string) bool {
		return a == b
	}

	e.funcs["strLt"] = func(a, b string) bool {
		return a < b
	}
}

// registerCryptoFunctions 注册加密与编码函数
func (e *Engine) registerCryptoFunctions() {
	// 哈希函数
	e.funcs["md5"] = func(s string) string {
		hash := md5.Sum([]byte(s))
		return hex.EncodeToString(hash[:])
	}

	e.funcs["sha1"] = func(s string) string {
		hash := sha1.Sum([]byte(s))
		return hex.EncodeToString(hash[:])
	}

	e.funcs["sha256"] = func(s string) string {
		hash := sha256.Sum256([]byte(s))
		return hex.EncodeToString(hash[:])
	}

	// 编码函数
	e.funcs["base64Encode"] = func(s string) string {
		return base64.StdEncoding.EncodeToString([]byte(s))
	}

	e.funcs["base64Decode"] = func(s string) string {
		data, err := base64.StdEncoding.DecodeString(s)
		if err != nil {
			return ""
		}
		return string(data)
	}

	e.funcs["hexEncode"] = func(s string) string {
		return hex.EncodeToString([]byte(s))
	}

	e.funcs["hexDecode"] = func(s string) string {
		data, err := hex.DecodeString(s)
		if err != nil {
			return ""
		}
		return string(data)
	}
}
