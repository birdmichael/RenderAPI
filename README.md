# RenderAPI

## 最新更新

RenderAPI 现已实现以下重要更新：

1. **钩子系统重构**：将钩子实现分离到独立文件中，提高可维护性
   - `custom_hook.go` - 包含自定义钩子实现
   - `js_hook.go` - JavaScript钩子实现
   - `cmd_hook.go` - 命令行钩子实现

2. **异步钩子支持**：所有钩子现在都支持同步和异步执行模式

3. **前置和后置钩子分离**：模板定义中钩子分为`beforeHooks`和`afterHooks`两部分

4. **改进的缓存系统**：支持配置TTL和自定义缓存键模式

5. **重试机制增强**：完善的重试策略，支持指数退避

---

RenderAPI 是一个强大的 Go 语言 HTTP 客户端库，专为模板驱动的 API 请求设计。它允许用户通过 JSON 模板定义 HTTP 请求，支持动态数据插入和请求转换。

## 主要特性

- 灵活的 HTTP 客户端，支持所有标准 HTTP 方法
- 强大的 JSON 模板引擎，支持动态数据注入
- 可扩展的钩子系统，用于请求/响应拦截和修改
- 内置丰富的模板函数库，增强模板处理能力
- 缓存机制提高性能
- 简单直观的 API 设计

## 安装

```bash
go get github.com/birdmichael/RenderAPI
```

## 基本用法

```go
package main

import (
	"fmt"
	"github.com/birdmichael/RenderAPI/pkg/client"
	"github.com/birdmichael/RenderAPI/pkg/hooks"
)

func main() {
	// 创建 HTTP 客户端
	c := client.NewClient("https://api.example.com", 10)
	
	// 添加认证令牌
	c.SetHeader("Authorization", "Bearer your-token")
	
	// 添加请求日志钩子
	c.AddBeforeRequestHook(hooks.LoggingHook)
	
	// 发送 GET 请求
	resp, err := c.Get("/users")
	if err != nil {
		fmt.Printf("请求失败: %v\n", err)
		return
	}
	
	// 读取响应
	body, err := client.ReadResponseBody(resp)
	if err != nil {
		fmt.Printf("读取响应失败: %v\n", err)
		return
	}
	
	fmt.Println(string(body))
}
```

## 使用 JSON 模板

```go
package main

import (
	"fmt"
	"github.com/birdmichael/RenderAPI/pkg/client"
)

func main() {
	c := client.NewClient("https://api.example.com", 10)
	
	// 定义 JSON 模板
	template := `{
		"user": {
			"name": "{{.name}}",
			"age": {{.age}},
			"email": "{{.email}}"
		}
	}`
	
	// 准备数据
	data := map[string]interface{}{
		"name":  "张三",
		"age":   30,
		"email": "zhangsan@example.com",
	}
	
	// 发送带有模板的 POST 请求
	resp, err := c.PostWithTemplate("/users", template, data)
	if err != nil {
		fmt.Printf("请求失败: %v\n", err)
		return
	}
	
	// 处理响应...
}
```

## 使用文件模板和数据

```go
package main

import (
	"fmt"
	"github.com/birdmichael/RenderAPI/pkg/client"
)

func main() {
	c := client.NewClient("https://api.example.com", 10)
	
	// 使用文件模板和数据文件发送请求
	resp, err := c.PostWithTemplateFile("/users", "templates/user.json", "data/user_data.json")
	if err != nil {
		fmt.Printf("请求失败: %v\n", err)
		return
	}
	
	// 处理响应...
}
```

## 内置模板函数

RenderAPI 的模板引擎内置了丰富的函数库，使模板操作更加灵活强大。以下是可用的内置函数分类：

### 字符串操作函数

| 函数名 | 说明 | 示例 |
|--------|------|------|
| `toUpper` | 转换为大写 | `{{ toUpper "hello" }}` => `"HELLO"` |
| `toLower` | 转换为小写 | `{{ toLower "HELLO" }}` => `"hello"` |
| `title` | 首字母大写 | `{{ title "hello world" }}` => `"Hello World"` |
| `trim` | 去除前后空格 | `{{ trim "  hello  " }}` => `"hello"` |
| `trimPrefix` | 去除前缀 | `{{ trimPrefix "hello world" "hello " }}` => `"world"` |
| `trimSuffix` | 去除后缀 | `{{ trimSuffix "hello world" " world" }}` => `"hello"` |
| `replace` | 替换字符串(有次数限制) | `{{ replace "hello" "l" "L" 1 }}` => `"heLlo"` |
| `replaceAll` | 替换全部 | `{{ replaceAll "hello" "l" "L" }}` => `"heLLo"` |
| `split` | 分割字符串 | `{{ split "a,b,c" "," }}` => `["a", "b", "c"]` |
| `join` | 连接字符串 | `{{ join (split "a,b,c" ",") "-" }}` => `"a-b-c"` |
| `contains` | 是否包含子串 | `{{ contains "hello" "ll" }}` => `true` |
| `hasPrefix` | 是否有前缀 | `{{ hasPrefix "hello" "he" }}` => `true` |
| `hasSuffix` | 是否有后缀 | `{{ hasSuffix "hello" "lo" }}` => `true` |
| `length` | 字符串长度 | `{{ length "hello" }}` => `5` |
| `regexMatch` | 正则匹配 | `{{ regexMatch "^h.*o$" "hello" }}` => `true` |
| `regexReplace` | 正则替换 | `{{ regexReplace "[aeiou]" "*" "hello" }}` => `"h*ll*"` |
| `urlEncode` | URL编码 | `{{ urlEncode "hello world" }}` => `"hello+world"` |
| `urlDecode` | URL解码 | `{{ urlDecode "hello+world" }}` => `"hello world"` |
| `htmlEscape` | HTML转义 | `{{ htmlEscape "<div>" }}` => `"&lt;div&gt;"` |
| `htmlUnescape` | HTML反转义 | `{{ htmlUnescape "&lt;div&gt;" }}` => `"<div>"` |
| `substr` | 子字符串 | `{{ substr "hello" 1 2 }}` => `"el"` |
| `repeat` | 重复字符串 | `{{ repeat "ab" 3 }}` => `"ababab"` |

### 日期时间函数

| 函数名 | 说明 | 示例 |
|--------|------|------|
| `now` | 当前时间 | `{{ now }}` => 当前时间对象 |
| `formatTime` | 格式化时间 | `{{ formatTime (now) "2006-01-02" }}` => 如 `"2023-01-01"` |
| `formatDate` | 格式化日期 | `{{ formatDate (now) }}` => 如 `"2023-01-01"` |
| `formatDateTime` | 格式化日期时间 | `{{ formatDateTime (now) }}` => 如 `"2023-01-01 12:34:56"` |
| `parseTime` | 解析时间字符串 | `{{ parseTime "2006-01-02" "2023-01-01" }}` => 时间对象 |
| `addDate` | 添加天数 | `{{ addDate (now) 7 }}` => 一周后的时间 |
| `addHours` | 添加小时 | `{{ addHours (now) 2 }}` => 两小时后的时间 |
| `addMinutes` | 添加分钟 | `{{ addMinutes (now) 30 }}` => 30分钟后的时间 |
| `since` | 计算时间差(自某时刻起) | `{{ since (parseTime "2006-01-02" "2023-01-01") }}` => 时间差 |
| `until` | 计算时间差(距某时刻) | `{{ until (parseTime "2006-01-02" "2023-01-01") }}` => 时间差 |
| `isAfter` | 是否在之后 | `{{ isAfter (now) (parseTime "2006-01-02" "2000-01-01") }}` => `true` |
| `isBefore` | 是否在之前 | `{{ isBefore (now) (parseTime "2006-01-02" "2030-01-01") }}` => `true` |
| `year` | 获取年份 | `{{ year (now) }}` => 如 `2023` |
| `month` | 获取月份 | `{{ month (now) }}` => 如 `"January"` |
| `day` | 获取日 | `{{ day (now) }}` => 如 `1` |
| `weekday` | 获取星期几 | `{{ weekday (now) }}` => 如 `"Monday"` |
| `unixTime` | 获取Unix时间戳 | `{{ unixTime (now) }}` => 如 `1672531200` |
| `fromUnixTime` | 从Unix时间戳创建时间 | `{{ fromUnixTime 1672531200 }}` => 对应的时间对象 |

### 数学运算函数

| 函数名 | 说明 | 示例 |
|--------|------|------|
| `add` | 加法 | `{{ add 1 2 }}` => `3` |
| `sub` | 减法 | `{{ sub 5 2 }}` => `3` |
| `mul` | 乘法 | `{{ mul 2 3 }}` => `6` |
| `div` | 除法 | `{{ div 6 2 }}` => `3` |
| `mod` | 取模 | `{{ mod 7 3 }}` => `1` |
| `ceil` | 向上取整 | `{{ ceil 3.2 }}` => `4` |
| `floor` | 向下取整 | `{{ floor 3.8 }}` => `3` |
| `round` | 四舍五入 | `{{ round 3.5 }}` => `4` |
| `max` | 最大值 | `{{ max 1 5 }}` => `5` |
| `min` | 最小值 | `{{ min 1 5 }}` => `1` |
| `abs` | 绝对值 | `{{ abs -5 }}` => `5` |
| `pow` | 幂运算 | `{{ pow 2 3 }}` => `8` |
| `sqrt` | 平方根 | `{{ sqrt 16 }}` => `4` |
| `rand` | 随机数(0-1) | `{{ rand }}` => 随机小数 |
| `randInt` | 随机整数 | `{{ randInt 1 10 }}` => 1到10间的随机整数 |

### 数据转换函数

| 函数名 | 说明 | 示例 |
|--------|------|------|
| `toString` | 转换为字符串 | `{{ toString 123 }}` => `"123"` |
| `toInt` | 转换为整数 | `{{ toInt "123" }}` => `123` |
| `toFloat` | 转换为浮点数 | `{{ toFloat "3.14" }}` => `3.14` |
| `toBool` | 转换为布尔值 | `{{ toBool "true" }}` => `true` |
| `jsonEncode` | JSON编码 | `{{ jsonEncode (dict "name" "张三") }}` => `{"name":"张三"}` |
| `jsonDecode` | JSON解码 | `{{ (jsonDecode "{\"name\":\"张三\"}").name }}` => `"张三"` |
| `prettifyJSON` | 美化JSON | `{{ prettifyJSON "{\"name\":\"张三\"}" }}` => 格式化后的JSON |

### 集合操作函数

| 函数名 | 说明 | 示例 |
|--------|------|------|
| `first` | 获取第一个元素 | `{{ first .items }}` => 集合第一个元素 |
| `last` | 获取最后一个元素 | `{{ last .items }}` => 集合最后一个元素 |
| `slice` | 切片 | `{{ slice .items 1 3 }}` => 索引1到3的元素 |
| `append` | 追加元素 | `{{ append .items "new" }}` => 添加元素后的集合 |
| `indexOf` | 查找索引 | `{{ indexOf .items "item" }}` => 元素在集合中的索引 |
| `reverse` | 反转集合 | `{{ reverse .items }}` => 反转后的集合 |
| `keys` | 获取Map的键 | `{{ keys .dict }}` => 所有键的切片 |
| `values` | 获取Map的值 | `{{ values .dict }}` => 所有值的切片 |
| `hasKey` | 是否有键 | `{{ hasKey .dict "name" }}` => 是否包含指定键 |
| `sum` | 求和 | `{{ sum .numbers }}` => 数组所有元素的和 |
| `avg` | 求平均值 | `{{ avg .numbers }}` => 数组元素的平均值 |

### 条件逻辑函数

| 函数名 | 说明 | 示例 |
|--------|------|------|
| `ternary` | 三元操作符 | `{{ ternary true "真" "假" }}` => `"真"` |
| `defaultValue` | 默认值 | `{{ defaultValue .name "默认名称" }}` => 当name为nil时返回默认值 |
| `coalesce` | 返回第一个非空值 | `{{ coalesce .name .nickname "匿名" }}` => 第一个非nil非空字符串值 |
| `and` | 逻辑与 | `{{ and true false }}` => `false` |
| `or` | 逻辑或 | `{{ or true false }}` => `true` |
| `not` | 逻辑非 | `{{ not true }}` => `false` |
| `eq` | 相等 | `{{ eq 5 5 }}` => `true` |
| `ne` | 不等 | `{{ ne 5 6 }}` => `true` |
| `lt` | 小于 | `{{ lt 5 10 }}` => `true` |
| `le` | 小于等于 | `{{ le 5 5 }}` => `true` |
| `gt` | 大于 | `{{ gt 10 5 }}` => `true` |
| `ge` | 大于等于 | `{{ ge 5 5 }}` => `true` |

### 加密与编码函数

| 函数名 | 说明 | 示例 |
|--------|------|------|
| `md5` | MD5哈希 | `{{ md5 "hello" }}` => MD5哈希字符串 |
| `sha1` | SHA1哈希 | `{{ sha1 "hello" }}` => SHA1哈希字符串 |
| `sha256` | SHA256哈希 | `{{ sha256 "hello" }}` => SHA256哈希字符串 |
| `base64Encode` | Base64编码 | `{{ base64Encode "hello" }}` => `"aGVsbG8="` |
| `base64Decode` | Base64解码 | `{{ base64Decode "aGVsbG8=" }}` => `"hello"` |
| `hexEncode` | 十六进制编码 | `{{ hexEncode "hello" }}` => `"68656c6c6f"` |
| `hexDecode` | 十六进制解码 | `{{ hexDecode "68656c6c6f" }}` => `"hello"` |

## 模板示例

以下是使用内置函数的模板示例：

### 基本数据处理

```json
{
  "user": {
    "name": "{{ toUpper .name }}",
    "email": "{{ toLower .email }}",
    "registered_at": "{{ formatDateTime (now) }}",
    "is_active": {{ toBool .status }},
    "subscription": {
      "type": "{{ defaultValue .plan "free" }}",
      "expires_in_days": {{ div (toFloat .expires_seconds) 86400 }}
    }
  },
  "security": {
    "token_hash": "{{ sha256 .token }}",
    "login_ip": "{{ regexReplace "(\\d+)\\.(\\d+)\\.(\\d+)\\.(\\d+)" "$1.$2.XXX.XXX" .ip }}"
  }
}
```

### 条件逻辑

```json
{
  "order": {
    "id": "{{ .order_id }}",
    "status": "{{ .status }}",
    "total": {{ .total }},
    "discount": {{ ternary (gt (toFloat .total) 1000) 0.15 0.05 }},
    "final_price": {{ sub (toFloat .total) (mul (toFloat .total) (ternary (gt (toFloat .total) 1000) 0.15 0.05)) }},
    "shipping": {
      "method": "{{ coalesce .shipping_method .default_shipping "standard" }}",
      "estimate_days": {{ ternary (eq .shipping_method "express") 1 (ternary (eq .shipping_method "priority") 3 7) }}
    }
  }
}
```

### 集合处理

```json
{
  "analytics": {
    "total_items": {{ length .items }},
    "categories": {{ keys .categories }},
    "most_expensive": {{ last (slice (sort .prices) 0 (len .prices)) }},
    "average_price": {{ avg .prices }},
    "price_summary": {
      "min": {{ min (index .prices 0) (index .prices 1) }},
      "max": {{ max (index .prices 0) (index .prices 1) }}
    }
  },
  "highlighted_item": {{ jsonEncode (first .featured_items) }}
}
```

## 钩子系统

RenderAPI提供了强大的钩子系统，支持前置钩子（BeforeHooks）和后置钩子（AfterHooks），允许你在请求前后进行拦截和修改。现在钩子系统还支持异步执行：

```go
// 添加请求日志钩子
client.AddBeforeHook(&hooks.LoggingHook{})

// 添加认证钩子
client.AddBeforeHook(hooks.NewAuthHook("your-token"))

// 添加响应日志钩子
client.AddAfterHook(&hooks.ResponseLogHook{})

// 添加字段转换钩子
transformMap := map[string]string{
    "user": "phone"  // 将 user 字段转换为 phone 字段
}
client.AddBeforeHook(hooks.NewFieldTransformHook(transformMap))

// 添加自定义钩子
client.AddBeforeHook(&hooks.CustomFunctionHook{
    BeforeFn: func(req *http.Request) (*http.Request, error) {
        req.Header.Set("X-Custom-Header", "value")
        return req, nil
    },
})
```

## JavaScript脚本钩子

你可以使用JavaScript脚本来动态修改请求和响应：

```go
// 从文件加载JavaScript脚本钩子
if err := client.AddJSHookFromFile("scripts/transform_request.js", false, 30); err != nil {
    log.Fatalf("加载脚本失败: %v", err)
}

// 从字符串加载JavaScript脚本钩子
scriptContent := `
function processRequest(request) {
    // 修改请求体
    var body = JSON.parse(request.body);
    body.timestamp = new Date().toISOString();
    request.body = JSON.stringify(body);
    return request;
}
`
if err := client.AddJSHookFromString(scriptContent, false, 30); err != nil {
    log.Fatalf("添加脚本钩子失败: %v", err)
}
```

JavaScript脚本示例:

```javascript
// scripts/transform_request.js
function processRequest(request) {
    // 读取请求体
    var body = JSON.parse(request.body);
    
    // 修改请求体
    if (body.user) {
        body.user.name = body.user.name.toUpperCase();
        body.user.created_at = new Date().toISOString();
        
        // 使用内置的RSA加密函数加密敏感信息
        if (body.user.password) {
            body.user.password = rsaEncryptGo(body.user.password, publicKey);
        }
    }
    
    // 返回修改后的请求体
    request.body = JSON.stringify(body);
    return request;
}
```

## 命令行钩子

你可以使用命令行脚本处理请求和响应：

```go
// 添加命令行钩子（非异步，30秒超时）
client.AddCommandHook("jq '.user.name = .user.name | ascii_upcase'", false, 30)
```

## 模板定义中的钩子

在模板定义文件中，你可以指定前置钩子和后置钩子：

```json
{
  "request": {
    "method": "POST",
    "baseURL": "https://api.example.com",
    "path": "/users",
    "headers": {
      "Content-Type": "application/json"
    }
  },
  "beforeHooks": [
    {
      "type": "js",
      "name": "authHook",
      "script": "function processRequest(request) { console.log('Auth处理请求...'); return request; }",
      "async": false,
      "timeout": 10
    },
    {
      "type": "command",
      "name": "timestamp",
      "command": "jq '.body.timestamp = now'",
      "async": false,
      "timeout": 3
    }
  ],
  "afterHooks": [
    {
      "type": "js",
      "name": "postProcess",
      "script": "function processResponse(response) { console.log('后置处理响应...'); return response; }",
      "async": false,
      "timeout": 5
    }
  ],
  "body": {
    "user": {
      "name": "{{.name}}",
      "email": "{{.email}}"
    }
  }
}
```

## 缓存系统

RenderAPI 提供了内置的缓存系统，可以提高性能并减少重复请求。在模板定义中配置缓存：

```json
{
  "request": {
    "method": "GET",
    "path": "/users/{{.user_id}}"
  },
  "caching": {
    "enabled": true,
    "ttl": 300,
    "keyPattern": "users-{{.user_id}}"
  }
}
```

缓存配置说明：
- `enabled`: 是否启用缓存
- `ttl`: 缓存的生存时间（秒）
- `keyPattern`: 可选的缓存键模式，支持模板语法。如果未指定，将使用请求URL和请求体的哈希作为键

## 重试机制

对于不稳定的API，RenderAPI提供了内置的重试机制：

```json
{
  "request": {
    "method": "POST",
    "path": "/process"
  },
  "retry": {
    "enabled": true,
    "maxAttempts": 3,
    "initialDelay": 1000,
    "backoffFactor": 2
  }
}
```

重试配置说明：
- `enabled`: 是否启用重试
- `maxAttempts`: 最大尝试次数
- `initialDelay`: 首次重试前的延迟（毫秒）
- `backoffFactor`: 退避因子，用于计算后续重试的延迟时间

## 项目结构

```
RenderAPI/
├── cmd/                # 命令行工具
│   └── httpclient/     # HTTP客户端命令行工具
├── pkg/                # 核心包
│   ├── client/         # HTTP客户端实现
│   ├── template/       # 模板引擎
│   ├── hooks/          # 请求/响应钩子
│   │   ├── hooks.go         # 钩子接口和通用功能
│   │   ├── custom_hook.go   # 自定义钩子实现
│   │   ├── js_hook.go       # JavaScript钩子实现
│   │   └── cmd_hook.go      # 命令行钩子实现
│   └── config/         # 配置管理
├── examples/           # 使用示例
│   ├── basic/          # 基本使用示例
│   ├── advanced/       # 高级功能示例
│   └── template_file/  # 模板文件示例
├── testdata/           # 测试数据
└── internal/           # 内部工具和辅助函数
    └── utils/          # 工具函数
```

## 测试

RenderAPI 包含详尽的单元测试和集成测试。运行以下命令来执行测试：

```bash
# 运行所有测试
make test

# 生成测试覆盖率报告
make test-coverage

# 运行基准测试
make bench
```

## 使用场景

RenderAPI 特别适用于以下场景：

1. **API 客户端开发**：轻松构建与复杂 API 交互的客户端
2. **自动化测试**：通过模板定义请求，简化 API 测试
3. **API 代理**：动态转换和处理 API 请求
4. **数据采集**：从多个 API 源收集和处理数据

## 贡献

欢迎提交Issues和Pull Requests！

## 许可证

本项目基于 MIT 许可证发布 - 详情请查看 [LICENSE](LICENSE) 文件。 