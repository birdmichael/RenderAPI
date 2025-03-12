// 该示例展示如何使用RenderAPI的内置模板函数
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/birdmichael/RenderAPI/pkg/template"
)

func main() {
	// 创建新的模板引擎
	engine := template.NewEngine()

	// 使用字符串操作函数的模板
	stringTemplate := `{
  "profile": {
    "username": "{{ toUpper .username }}",
    "display_name": "{{ title .name }}",
    "email": "{{ toLower .email }}",
    "bio": "{{ trim .bio }}",
    "url_safe_username": "{{ urlEncode .username }}",
    "html_bio": "{{ htmlEscape .htmlBio }}",
    "admin_role": "{{ index .roles 1 }}"
  }
}`

	// 使用日期和数学函数的模板
	mathTemplate := `{
  "calculation": {
    "num1": {{ .num1 }},
    "num2": {{ .num2 }},
    "add": {{ add .num1 .num2 }},
    "subtract": {{ sub .num1 .num2 }},
    "multiply": {{ mul .num1 .num2 }},
    "divide": {{ div .num1 .num2 }},
    "max": {{ max .num1 .num2 }},
    "min": {{ min .num1 .num2 }},
    "round": {{ round .float }},
    "ceil": {{ ceil .float }},
    "floor": {{ floor .float }},
    "random": {{ randInt 1 100 }}
  }
}`

	// 使用条件函数的模板
	conditionTemplate := `{
  "user": {
    "name": "{{ .name }}",
    "nickname": "{{ defaultValue .nickname "匿名用户" }}",
    "status": "{{ ternary .active "活跃" "不活跃" }}",
    "access_level": "{{ coalesce .level .default_level "普通用户" }}",
    "premium": {{ and .active .premium }}
  }
}`

	// 添加模板
	err := engine.AddTemplate("string-template", stringTemplate)
	if err != nil {
		log.Fatalf("添加字符串模板失败: %v", err)
	}

	err = engine.AddTemplate("math-template", mathTemplate)
	if err != nil {
		log.Fatalf("添加数学模板失败: %v", err)
	}

	err = engine.AddTemplate("condition-template", conditionTemplate)
	if err != nil {
		log.Fatalf("添加条件模板失败: %v", err)
	}

	// 准备字符串操作示例数据
	stringData := map[string]interface{}{
		"username": "john_doe",
		"name":     "john doe",
		"email":    "JOHN.DOE@EXAMPLE.COM",
		"bio":      "  这是一个简介，包含了一些信息  ",
		"htmlBio":  "<div>这是HTML内容</div>",
		"roles":    []string{"user", "admin", "editor"},
	}

	// 准备数学示例数据
	mathData := map[string]interface{}{
		"num1":  10.0,
		"num2":  5.0,
		"float": 3.75,
	}

	// 准备条件示例数据
	conditionData := map[string]interface{}{
		"name":          "张三",
		"nickname":      "",
		"active":        true,
		"premium":       true,
		"level":         nil,
		"default_level": "高级用户",
	}

	// 执行字符串模板
	stringResult, err := engine.Execute("string-template", stringData)
	if err != nil {
		log.Fatalf("执行字符串模板失败: %v", err)
	}

	// 执行数学模板
	mathResult, err := engine.Execute("math-template", mathData)
	if err != nil {
		log.Fatalf("执行数学模板失败: %v", err)
	}

	// 执行条件模板
	conditionResult, err := engine.Execute("condition-template", conditionData)
	if err != nil {
		log.Fatalf("执行条件模板失败: %v", err)
	}

	// 格式化输出结果
	stringJson := formatJSON(stringResult)
	mathJson := formatJSON(mathResult)
	conditionJson := formatJSON(conditionResult)

	// 输出结果
	fmt.Println("=== 字符串操作示例 ===")
	fmt.Println(stringJson)
	fmt.Println()

	fmt.Println("=== 数学运算示例 ===")
	fmt.Println(mathJson)
	fmt.Println()

	fmt.Println("=== 条件逻辑示例 ===")
	fmt.Println(conditionJson)
	fmt.Println()

	// 保存输出到文件
	err = saveToFile("string_output.json", stringJson)
	if err != nil {
		log.Printf("保存字符串输出失败: %v", err)
	}

	err = saveToFile("math_output.json", mathJson)
	if err != nil {
		log.Printf("保存数学输出失败: %v", err)
	}

	err = saveToFile("condition_output.json", conditionJson)
	if err != nil {
		log.Printf("保存条件输出失败: %v", err)
	}

	fmt.Println("所有示例已运行完成，输出已保存到JSON文件")
}

// 格式化JSON
func formatJSON(data string) string {
	var out bytes.Buffer
	err := json.Indent(&out, []byte(data), "", "  ")
	if err != nil {
		return data
	}
	return out.String()
}

// 保存到文件
func saveToFile(filename, content string) error {
	return os.WriteFile(filename, []byte(content), 0644)
}
