package template

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestNewEngine 测试创建模板引擎
func TestNewEngine(t *testing.T) {
	engine := NewEngine()
	if engine == nil {
		t.Fatal("创建模板引擎失败")
	}

	if engine.templates == nil {
		t.Error("模板映射未初始化")
	}

	if engine.funcs == nil {
		t.Error("函数映射未初始化")
	}

	if engine.cache == nil {
		t.Error("缓存映射未初始化")
	}
}

// TestAddFunc 测试添加自定义函数
func TestAddFunc(t *testing.T) {
	engine := NewEngine()

	// 添加自定义函数
	engine.AddFunc("multiply", func(a, b int) int {
		return a * b
	})

	// 添加模板，使用自定义函数
	tmplStr := `结果: {{multiply 6 7}}`
	err := engine.AddTemplate("test-func", tmplStr)
	if err != nil {
		t.Fatalf("添加模板失败: %v", err)
	}

	// 执行模板
	result, err := engine.Execute("test-func", nil)
	if err != nil {
		t.Fatalf("执行模板失败: %v", err)
	}

	// 验证结果
	expected := "结果: 42"
	if result != expected {
		t.Errorf("结果错误，期望: %s, 实际: %s", expected, result)
	}
}

// TestAddTemplate 测试添加模板
func TestAddTemplate(t *testing.T) {
	engine := NewEngine()

	// 添加简单模板
	tmplStr := "Hello, {{.Name}}!"
	err := engine.AddTemplate("test-simple", tmplStr)
	if err != nil {
		t.Fatalf("添加模板失败: %v", err)
	}

	// 验证模板是否存在
	_, exists := engine.GetTemplate("test-simple")
	if !exists {
		t.Error("无法获取添加的模板")
	}

	// 添加相同名称的模板应该覆盖
	newTmplStr := "Welcome, {{.Name}}!"
	err = engine.AddTemplate("test-simple", newTmplStr)
	if err != nil {
		t.Fatalf("覆盖模板失败: %v", err)
	}

	// 验证模板已更新
	tmpl, exists := engine.GetTemplate("test-simple")
	if !exists {
		t.Error("无法获取覆盖后的模板")
	}

	var buf strings.Builder
	err = tmpl.Execute(&buf, map[string]string{"Name": "世界"})
	if err != nil {
		t.Fatalf("执行模板失败: %v", err)
	}

	expected := "Welcome, 世界!"
	if buf.String() != expected {
		t.Errorf("结果错误，期望: %s, 实际: %s", expected, buf.String())
	}

	// 添加无效模板
	err = engine.AddTemplate("test-invalid", "Hello, {{.Name")
	if err == nil {
		t.Error("应该检测到无效模板")
	}
}

// TestHasTemplate 测试检查模板是否存在
func TestHasTemplate(t *testing.T) {
	engine := NewEngine()

	// 添加模板
	err := engine.AddTemplate("test-exists", "Hello, {{.Name}}!")
	if err != nil {
		t.Fatalf("添加模板失败: %v", err)
	}

	// 检查存在的模板
	if !engine.HasTemplate("test-exists") {
		t.Error("HasTemplate未能检测到存在的模板")
	}

	// 检查不存在的模板
	if engine.HasTemplate("test-not-exists") {
		t.Error("HasTemplate错误地报告了不存在的模板")
	}
}

// TestRemoveTemplate 测试删除模板
func TestRemoveTemplate(t *testing.T) {
	engine := NewEngine()

	// 添加模板
	err := engine.AddTemplate("test-to-remove", "Hello, {{.Name}}!")
	if err != nil {
		t.Fatalf("添加模板失败: %v", err)
	}

	// 确认模板存在
	if !engine.HasTemplate("test-to-remove") {
		t.Error("模板未成功添加")
	}

	// 删除模板
	engine.RemoveTemplate("test-to-remove")

	// 确认模板已删除
	if engine.HasTemplate("test-to-remove") {
		t.Error("模板未被成功删除")
	}
}

// TestExecute 测试执行模板
func TestExecute(t *testing.T) {
	engine := NewEngine()

	// 添加并执行简单模板
	t.Run("简单模板", func(t *testing.T) {
		err := engine.AddTemplate("test-execute", "Hello, {{.Name}}!")
		if err != nil {
			t.Fatalf("添加模板失败: %v", err)
		}

		result, err := engine.Execute("test-execute", map[string]string{"Name": "世界"})
		if err != nil {
			t.Fatalf("执行模板失败: %v", err)
		}

		expected := "Hello, 世界!"
		if result != expected {
			t.Errorf("结果错误，期望: %s, 实际: %s", expected, result)
		}
	})

	// 执行不存在的模板
	t.Run("不存在的模板", func(t *testing.T) {
		_, err := engine.Execute("non-existent", nil)
		if err == nil {
			t.Error("应该检测到不存在的模板")
		}
		if !strings.Contains(err.Error(), "找不到模板") {
			t.Errorf("错误消息不正确: %v", err)
		}
	})

	// 添加并执行带条件和循环的复杂模板
	t.Run("复杂模板", func(t *testing.T) {
		tmplStr := `姓名: {{.Name}}
{{if .IsAdmin}}管理员: 是{{else}}管理员: 否{{end}}
角色:{{range .Roles}}
 - {{.}}{{end}}`

		err := engine.AddTemplate("test-complex", tmplStr)
		if err != nil {
			t.Fatalf("添加模板失败: %v", err)
		}

		data := map[string]interface{}{
			"Name":    "测试用户",
			"IsAdmin": true,
			"Roles":   []string{"用户", "开发者", "测试者"},
		}

		result, err := engine.Execute("test-complex", data)
		if err != nil {
			t.Fatalf("执行模板失败: %v", err)
		}

		expected := `姓名: 测试用户
管理员: 是
角色:
 - 用户
 - 开发者
 - 测试者`

		if result != expected {
			t.Errorf("结果错误，期望:\n%s\n实际:\n%s", expected, result)
		}
	})
}

// TestRenderJSONTemplate 测试渲染JSON模板
func TestRenderJSONTemplate(t *testing.T) {
	engine := NewEngine()

	// 添加JSON模板
	tmplStr := `{
	"name": "{{.Name}}",
	"age": {{.Age}},
	"email": "{{.Email}}",
	"address": {
		"city": "{{.Address.City}}",
		"country": "{{.Address.Country}}"
	},
	"tags": [{{range $index, $tag := .Tags}}{{if $index}}, {{end}}"{{$tag}}"{{end}}]
}`

	err := engine.AddTemplate("user-json", tmplStr)
	if err != nil {
		t.Fatalf("添加模板失败: %v", err)
	}

	// 准备数据
	data := map[string]interface{}{
		"Name":  "张三",
		"Age":   30,
		"Email": "zhangsan@example.com",
		"Address": map[string]string{
			"City":    "北京",
			"Country": "中国",
		},
		"Tags": []string{"开发", "测试", "运维"},
	}

	// 渲染JSON模板
	result, err := engine.RenderJSONTemplate("user-json", data)
	if err != nil {
		t.Fatalf("渲染JSON模板失败: %v", err)
	}

	// 验证JSON结果
	var resultObj map[string]interface{}
	err = json.Unmarshal(result, &resultObj)
	if err != nil {
		t.Fatalf("解析结果JSON失败: %v", err)
	}

	// 验证基本字段
	if resultObj["name"] != "张三" || resultObj["age"] != float64(30) || resultObj["email"] != "zhangsan@example.com" {
		t.Errorf("基本字段错误: %v", resultObj)
	}

	// 验证嵌套对象
	address, ok := resultObj["address"].(map[string]interface{})
	if !ok {
		t.Fatalf("address不是对象: %v", resultObj["address"])
	}
	if address["city"] != "北京" || address["country"] != "中国" {
		t.Errorf("嵌套对象错误: %v", address)
	}

	// 验证数组
	tags, ok := resultObj["tags"].([]interface{})
	if !ok {
		t.Fatalf("tags不是数组: %v", resultObj["tags"])
	}
	if len(tags) != 3 || tags[0] != "开发" || tags[1] != "测试" || tags[2] != "运维" {
		t.Errorf("数组错误: %v", tags)
	}
}

// TestParseAndRenderJSON 测试直接解析和渲染JSON
func TestParseAndRenderJSON(t *testing.T) {
	engine := NewEngine()

	// 定义模板字符串
	tmplStr := `{"name": "{{.Name}}", "age": {{.Age}}}`

	// 准备数据
	data := map[string]interface{}{
		"Name": "李四",
		"Age":  25,
	}

	// 直接解析和渲染
	result, err := engine.ParseAndRenderJSON(tmplStr, data)
	if err != nil {
		t.Fatalf("解析和渲染JSON失败: %v", err)
	}

	// 验证结果
	var resultObj map[string]interface{}
	err = json.Unmarshal(result, &resultObj)
	if err != nil {
		t.Fatalf("解析结果JSON失败: %v", err)
	}

	if resultObj["name"] != "李四" || resultObj["age"] != float64(25) {
		t.Errorf("结果错误: %v", resultObj)
	}
}

// TestFormatJSON 测试格式化JSON
func TestFormatJSON(t *testing.T) {
	engine := NewEngine()

	// 未格式化的JSON
	unformatted := []byte(`{"name":"王五","nested":{"value":42},"array":[1,2,3]}`)

	// 格式化JSON
	formatted, err := engine.FormatJSON(unformatted)
	if err != nil {
		t.Fatalf("格式化JSON失败: %v", err)
	}

	// 验证格式化后的JSON
	expected := `{
  "array": [
    1,
    2,
    3
  ],
  "name": "王五",
  "nested": {
    "value": 42
  }
}`

	// 将结果规范化以便比较（忽略字段顺序）
	var expectedObj, formattedObj interface{}
	err = json.Unmarshal([]byte(expected), &expectedObj)
	if err != nil {
		t.Fatalf("解析期望JSON失败: %v", err)
	}
	err = json.Unmarshal(formatted, &formattedObj)
	if err != nil {
		t.Fatalf("解析结果JSON失败: %v", err)
	}

	// 重新序列化以便比较（忽略字段顺序）
	expectedBytes, _ := json.Marshal(expectedObj)
	formattedBytes, _ := json.Marshal(formattedObj)

	if string(expectedBytes) != string(formattedBytes) {
		t.Errorf("格式化结果与期望不匹配。\n期望: %s\n实际: %s", expected, string(formatted))
	}
}

// TestValidateJSON 测试验证JSON
func TestValidateJSON(t *testing.T) {
	engine := NewEngine()

	// 有效的JSON
	validJSON := []byte(`{"name": "赵六", "age": 35}`)
	err := engine.ValidateJSON(validJSON)
	if err != nil {
		t.Errorf("验证有效JSON失败: %v", err)
	}

	// 无效的JSON
	invalidJSON := []byte(`{"name": "赵六", "age": }`)
	err = engine.ValidateJSON(invalidJSON)
	if err == nil {
		t.Error("未能检测到无效JSON")
	}
}

// TestCacheWithSameData 测试使用相同数据的缓存
func TestCacheWithSameData(t *testing.T) {
	engine := NewEngine()

	// 添加模板
	tmplStr := `{"name": "{{.Name}}", "value": {{.Value}}}`
	err := engine.AddTemplate("cache-test", tmplStr)
	if err != nil {
		t.Fatalf("添加模板失败: %v", err)
	}

	// 准备数据
	data := map[string]interface{}{
		"Name":  "缓存测试",
		"Value": 100,
	}

	// 首次渲染
	result1, err := engine.RenderJSONTemplate("cache-test", data)
	if err != nil {
		t.Fatalf("首次渲染失败: %v", err)
	}

	// 再次渲染相同数据
	result2, err := engine.RenderJSONTemplate("cache-test", data)
	if err != nil {
		t.Fatalf("再次渲染失败: %v", err)
	}

	// 检查两次结果是否相同
	if string(result1) != string(result2) {
		t.Error("使用相同数据的两次渲染结果不同")
	}
}

// TestCacheClear 测试清除缓存
func TestCacheClear(t *testing.T) {
	engine := NewEngine()

	// 添加模板
	tmplStr := `{"name": "{{.Name}}", "value": {{.Value}}}`
	err := engine.AddTemplate("clear-cache-test", tmplStr)
	if err != nil {
		t.Fatalf("添加模板失败: %v", err)
	}

	// 准备数据
	data := map[string]interface{}{
		"Name":  "清除缓存测试",
		"Value": 200,
	}

	// 渲染以填充缓存
	_, err = engine.RenderJSONTemplate("clear-cache-test", data)
	if err != nil {
		t.Fatalf("渲染失败: %v", err)
	}

	// 清除缓存
	engine.ClearCache()

	// 验证缓存已清除
	if len(engine.cache) != 0 {
		t.Errorf("缓存未被清除，仍有 %d 个条目", len(engine.cache))
	}
}

func TestBuiltinFunctions(t *testing.T) {
	engine := NewEngine()

	testCases := []struct {
		name     string
		template string
		data     interface{}
		expected string
	}{
		// 字符串操作函数测试
		{
			name:     "字符串大小写转换",
			template: "{{ toUpper .str }} {{ toLower .str }} {{ title .str }}",
			data:     map[string]interface{}{"str": "Hello World"},
			expected: "HELLO WORLD hello world Hello World",
		},
		{
			name:     "字符串修剪",
			template: "{{ trim .str }}|{{ trimPrefix .str \"Hello\" }}|{{ trimSuffix .str \"World\" }}",
			data:     map[string]interface{}{"str": "  Hello World  "},
			expected: "Hello World|  Hello World  |  Hello World  ",
		},
		{
			name:     "字符串替换",
			template: "{{ replace .str \"l\" \"L\" 2 }}|{{ replaceAll .str \"l\" \"L\" }}",
			data:     map[string]interface{}{"str": "Hello World"},
			expected: "HeLLo World|HeLLo WorLd",
		},
		{
			name:     "字符串分割与连接",
			template: "{{ join (split .str \" \") \"-\" }}",
			data:     map[string]interface{}{"str": "Hello World"},
			expected: "Hello-World",
		},
		{
			name:     "字符串包含检查",
			template: "{{ contains .str \"World\" }}|{{ hasPrefix .str \"Hello\" }}|{{ hasSuffix .str \"World\" }}",
			data:     map[string]interface{}{"str": "Hello World"},
			expected: "true|true|true",
		},
		{
			name:     "字符串长度",
			template: "{{ length .str }}",
			data:     map[string]interface{}{"str": "Hello World"},
			expected: "11",
		},
		{
			name:     "正则表达式",
			template: "{{ regexMatch \"^H.*d$\" .str }}|{{ regexReplace \"[aeiou]\" \"*\" .str }}",
			data:     map[string]interface{}{"str": "Hello World"},
			expected: "true|H*ll* W*rld",
		},
		{
			name:     "URL编码解码",
			template: "{{ urlEncode .str }}|{{ urlDecode (urlEncode .str) }}",
			data:     map[string]interface{}{"str": "Hello World?"},
			expected: "Hello+World%3F|Hello World?",
		},
		{
			name:     "HTML转义",
			template: "{{ htmlEscape .str }}|{{ htmlUnescape (htmlEscape .str) }}",
			data:     map[string]interface{}{"str": "<div>Hello & World</div>"},
			expected: "&lt;div&gt;Hello &amp; World&lt;/div&gt;|<div>Hello & World</div>",
		},
		{
			name:     "子字符串",
			template: "{{ substr .str 6 5 }}",
			data:     map[string]interface{}{"str": "Hello World"},
			expected: "World",
		},
		{
			name:     "重复字符串",
			template: "{{ repeat .str 3 }}",
			data:     map[string]interface{}{"str": "ab"},
			expected: "ababab",
		},

		// 数学运算函数测试
		{
			name:     "基本运算",
			template: "{{ add .a .b }}|{{ sub .a .b }}|{{ mul .a .b }}|{{ div .a .b }}|{{ mod .a .b }}",
			data:     map[string]interface{}{"a": 10.0, "b": 3.0},
			expected: "13|7|30|3.3333333333333335|1",
		},
		{
			name:     "取整",
			template: "{{ ceil .a }}|{{ floor .a }}|{{ round .a }}",
			data:     map[string]interface{}{"a": 3.6},
			expected: "4|3|4",
		},
		{
			name:     "最大最小值",
			template: "{{ max .a .b }}|{{ min .a .b }}",
			data:     map[string]interface{}{"a": 10.0, "b": 3.0},
			expected: "10|3",
		},
		{
			name:     "绝对值",
			template: "{{ abs .a }}",
			data:     map[string]interface{}{"a": -5.5},
			expected: "5.5",
		},
		{
			name:     "幂运算",
			template: "{{ pow .a 2 }}|{{ sqrt .a }}",
			data:     map[string]interface{}{"a": 16.0},
			expected: "256|4",
		},

		// 类型转换函数测试
		{
			name:     "类型转换",
			template: "{{ toString .num }}|{{ toInt .str }}|{{ toFloat .int }}|{{ toBool .intBool }}",
			data:     map[string]interface{}{"num": 123, "str": "456", "int": 789, "intBool": 1},
			expected: "123|456|789|true",
		},
		{
			name:     "JSON操作",
			template: "{{ jsonEncode .obj }}|{{ (jsonDecode .jsonStr).name }}",
			data: map[string]interface{}{
				"obj":     map[string]interface{}{"name": "John", "age": 30},
				"jsonStr": `{"name":"Jane","age":25}`,
			},
			expected: `{"age":30,"name":"John"}|Jane`,
		},

		// 集合操作函数测试
		{
			name:     "数组函数",
			template: "{{ first .arr }}|{{ last .arr }}|{{ index (slice .arr 1 3) 0 }}",
			data:     map[string]interface{}{"arr": []interface{}{1, 2, 3, 4, 5}},
			expected: "1|5|2",
		},
		{
			name:     "Map函数",
			template: "{{ index (keys .map) 0 }}|{{ index (values .map) 1 }}|{{ hasKey .map \"name\" }}",
			data: map[string]interface{}{
				"map": map[string]interface{}{"name": "John", "age": 30},
			},
			expected: "age|30|true",
		},

		// 条件逻辑函数测试
		{
			name:     "条件选择",
			template: "{{ ternary .cond \"真\" \"假\" }}|{{ defaultValue .nil \"默认值\" }}|{{ coalesce .nil \"\" .value }}",
			data: map[string]interface{}{
				"cond":  true,
				"nil":   nil,
				"value": "有值",
			},
			expected: "真|默认值|有值",
		},
		{
			name:     "逻辑操作",
			template: "{{ and .a .b }}|{{ or .a .b }}|{{ not .a }}",
			data:     map[string]interface{}{"a": true, "b": false},
			expected: "false|true|false",
		},
		{
			name:     "比较操作",
			template: "{{ eq .a .b }}|{{ ne .a .b }}|{{ lt (toFloat .a) (toFloat .c) }}|{{ le (toFloat .a) (toFloat .c) }}|{{ gt (toFloat .a) (toFloat .c) }}|{{ ge (toFloat .c) (toFloat .a) }}",
			data:     map[string]interface{}{"a": 5, "b": 5, "c": 10},
			expected: "true|false|true|true|false|true",
		},

		// 加密与编码函数测试
		{
			name:     "哈希函数",
			template: "{{ md5 .str }}|{{ sha1 .str }}|{{ sha256 .str }}",
			data:     map[string]interface{}{"str": "test"},
			expected: "098f6bcd4621d373cade4e832627b4f6|a94a8fe5ccb19ba61c4c0873d391e987982fbbd3|9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08",
		},
		{
			name:     "编码函数",
			template: "{{ $encoded := base64Encode .str }}{{ $encoded }}|{{ base64Decode $encoded }}",
			data:     map[string]interface{}{"str": "Hello World"},
			expected: "SGVsbG8gV29ybGQ=|Hello World",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tmplName := "test_builtin_" + tc.name
			err := engine.AddTemplate(tmplName, tc.template)
			if err != nil {
				t.Fatalf("添加模板失败: %v", err)
			}

			result, err := engine.Execute(tmplName, tc.data)
			if err != nil {
				t.Fatalf("执行模板失败: %v", err)
			}

			if string(result) != tc.expected {
				t.Errorf("期望: %q, 实际: %q", tc.expected, string(result))
			}
		})
	}
}
