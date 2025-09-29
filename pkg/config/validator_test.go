package config

import (
	"fmt"
	"testing"
)

func TestConfigValidator_Basic(t *testing.T) {
	validator := NewConfigValidator()
	
	if validator == nil {
		t.Fatal("NewConfigValidator 返回 nil")
	}
	
	if len(validator.rules) != 0 {
		t.Errorf("期望初始规则数量为0，实际为 %d", len(validator.rules))
	}
}

func TestConfigValidator_AddRule(t *testing.T) {
	validator := NewConfigValidator()
	
	rule := ValidationRule{
		Field:       "test.field",
		Required:    true,
		Description: "测试字段",
		Validator: func(value interface{}) error {
			return nil
		},
	}
	
	validator.AddRule(rule)
	
	if len(validator.rules) != 1 {
		t.Errorf("期望规则数量为1，实际为 %d", len(validator.rules))
	}
	
	if validator.rules[0].Field != "test.field" {
		t.Errorf("期望字段为 test.field，实际为 %s", validator.rules[0].Field)
	}
}

func TestConfigValidator_Validate(t *testing.T) {
	validator := NewConfigValidator()
	
	// 添加测试规则
	validator.AddRule(ValidationRule{
		Field:       "app.name",
		Required:    true,
		Description: "应用名称",
		Validator: func(value interface{}) error {
			name, ok := value.(string)
			if !ok {
				return fmt.Errorf("应用名称必须是字符串")
			}
			if len(name) < 3 {
				return fmt.Errorf("应用名称长度必须大于等于3")
			}
			return nil
		},
	})
	
	validator.AddRule(ValidationRule{
		Field:       "app.version",
		Required:    false,
		Description: "应用版本",
		Validator: func(value interface{}) error {
			version, ok := value.(string)
			if !ok {
				return fmt.Errorf("应用版本必须是字符串")
			}
			if version != "" && len(version) < 3 {
				return fmt.Errorf("应用版本长度必须大于等于3")
			}
			return nil
		},
	})
	
	tests := []struct {
		name      string
		config    *Config
		wantValid bool
		errCount  int
	}{
		{
			name: "有效配置",
			config: &Config{
				App: AppConfig{
					Name:    "TestApp",
					Version: "v1.0.0",
				},
			},
			wantValid: true,
			errCount:  0,
		},
		{
			name: "名称为空",
			config: &Config{
				App: AppConfig{
					Name:    "",
					Version: "v1.0.0",
				},
			},
			wantValid: false,
			errCount:  1,
		},
		{
			name: "名称太短",
			config: &Config{
				App: AppConfig{
					Name:    "Te",
					Version: "v1.0.0",
				},
			},
			wantValid: false,
			errCount:  1,
		},
		{
			name: "版本可选且有效",
			config: &Config{
				App: AppConfig{
					Name:    "TestApp",
					Version: "v2.0.0",
				},
			},
			wantValid: true,
			errCount:  0,
		},
		{
			name: "版本可选但无效",
			config: &Config{
				App: AppConfig{
					Name:    "TestApp",
					Version: "v2",
				},
			},
			wantValid: false,
			errCount:  1,
		},
		{
			name: "版本为空（可选字段）",
			config: &Config{
				App: AppConfig{
					Name:    "TestApp",
					Version: "",
				},
			},
			wantValid: true,
			errCount:  0,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.Validate(tt.config)
			
			if result.IsValid != tt.wantValid {
				t.Errorf("期望验证结果为 %v，实际为 %v", tt.wantValid, result.IsValid)
			}
			
			if len(result.Errors) != tt.errCount {
				t.Errorf("期望错误数量为 %d，实际为 %d", tt.errCount, len(result.Errors))
				for i, err := range result.Errors {
					t.Logf("错误 %d: Field=%s, Message=%s", i, err.Field, err.Message)
				}
			}
		})
	}
}

func TestConfigValidator_GetFieldValue(t *testing.T) {
	validator := NewConfigValidator()
	
	config := &Config{
		Bot: BotConfig{
			Token: "test_token",
			Debug: true,
			Webhook: WebhookConfig{
				Enabled: true,
				URL:     "https://example.com",
				Port:    8443,
			},
		},
		Database: DatabaseConfig{
			Driver:       "sqlite3",
			DSN:          "./test.db",
			MaxOpenConns: 25,
			MaxIdleConns: 10,
		},
		Server: ServerConfig{
			Port: "8080",
			Host: "localhost",
		},
		App: AppConfig{
			Name:        "TestApp",
			Version:     "v1.0.0",
			Environment: "testing",
		},
	}
	
	tests := []struct {
		name     string
		field    string
		expected interface{}
	}{
		{
			name:     "Bot Token",
			field:    "bot.token",
			expected: "test_token",
		},
		{
			name:     "Bot Debug",
			field:    "bot.debug",
			expected: true,
		},
		{
			name:     "Webhook Enabled",
			field:    "bot.webhook.enabled",
			expected: true,
		},
		{
			name:     "Webhook Port",
			field:    "bot.webhook.port",
			expected: 8443,
		},
		{
			name:     "Database Driver",
			field:    "database.driver",
			expected: "sqlite3",
		},
		{
			name:     "Database MaxOpenConns",
			field:    "database.max_open_conns",
			expected: 25,
		},
		{
			name:     "Server Port",
			field:    "server.port",
			expected: "8080",
		},
		{
			name:     "App Name",
			field:    "app.name",
			expected: "TestApp",
		},
		{
			name:     "App Environment",
			field:    "app.environment",
			expected: "testing",
		},
		{
			name:     "无效字段",
			field:    "invalid.field",
			expected: nil,
		},
		{
			name:     "无效嵌套字段",
			field:    "bot.invalid",
			expected: nil,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.getFieldValue(config, tt.field)
			if result != tt.expected {
				t.Errorf("期望 %v，实际 %v", tt.expected, result)
			}
		})
	}
}

func TestConfigValidator_IsEmpty(t *testing.T) {
	validator := NewConfigValidator()
	
	tests := []struct {
		name     string
		value    interface{}
		expected bool
	}{
		{
			name:     "nil值",
			value:    nil,
			expected: true,
		},
		{
			name:     "空字符串",
			value:    "",
			expected: true,
		},
		{
			name:     "空白字符串",
			value:    "   ",
			expected: true,
		},
		{
			name:     "非空字符串",
			value:    "hello",
			expected: false,
		},
		{
			name:     "零整数",
			value:    0,
			expected: true,
		},
		{
			name:     "非零整数",
			value:    42,
			expected: false,
		},
		{
			name:     "零浮点数",
			value:    0.0,
			expected: true,
		},
		{
			name:     "非零浮点数",
			value:    3.14,
			expected: false,
		},
		{
			name:     "false布尔值",
			value:    false,
			expected: false, // bool类型不为空
		},
		{
			name:     "true布尔值",
			value:    true,
			expected: false,
		},
		{
			name:     "其他类型",
			value:    struct{}{},
			expected: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.isEmpty(tt.value)
			if result != tt.expected {
				t.Errorf("期望 %v，实际 %v", tt.expected, result)
			}
		})
	}
}

func TestGetDefaultValidator(t *testing.T) {
	validator := GetDefaultValidator()
	
	if validator == nil {
		t.Fatal("GetDefaultValidator 返回 nil")
	}
	
	// 验证默认规则数量（应该包含所有标准配置项的验证规则）
	if len(validator.rules) < 10 {
		t.Errorf("期望默认验证器包含至少10条规则，实际为 %d", len(validator.rules))
	}
	
	// 测试默认验证器对有效配置的验证
	validConfig := &Config{
		Bot: BotConfig{
			Token: "8479463724:AAHzlbx-9qMUarK5BMG9hAnbAzvCQT1IJ9g",
		},
		Database: DatabaseConfig{
			Driver:       "sqlite3",
			DSN:          "./data/test.db",
			MaxOpenConns: 25,
			MaxIdleConns: 10,
		},
		Server: ServerConfig{
			Port: "8080",
			Host: "0.0.0.0",
		},
		Scheduler: SchedulerConfig{
			Timezone:   "Asia/Shanghai",
			MaxWorkers: 10,
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "json",
			Output: "stdout",
		},
		App: AppConfig{
			Name:        "MMemory",
			Version:     "v0.0.1",
			Environment: "development",
		},
		Monitoring: MonitoringConfig{
			Enabled: true,
			Port:    9090,
			Path:    "/metrics",
		},
	}
	
	result := validator.Validate(validConfig)
	if !result.IsValid {
		t.Errorf("默认验证器应该验证有效配置为有效，错误: %v", result.Errors)
	}
}

func TestConfigValidator_ComplexValidation(t *testing.T) {
	validator := NewConfigValidator()
	
	// 添加复杂的验证规则
	validator.AddRule(ValidationRule{
		Field:       "database.max_open_conns",
		Required:    true,
		Description: "数据库最大连接数",
		Validator: func(value interface{}) error {
			conns, ok := value.(int)
			if !ok {
				return fmt.Errorf("连接数必须是整数")
			}
			if conns <= 0 {
				return fmt.Errorf("最大连接数必须大于0")
			}
			if conns > 1000 {
				return fmt.Errorf("最大连接数不能超过1000")
			}
			return nil
		},
	})
	
	validator.AddRule(ValidationRule{
		Field:       "server.port",
		Required:    true,
		Description: "服务器端口",
		Validator: func(value interface{}) error {
			_, ok := value.(string)
			if !ok {
				return fmt.Errorf("端口必须是字符串")
			}
			// 这里可以添加更复杂的端口验证逻辑
			return nil
		},
	})
	
	tests := []struct {
		name      string
		config    *Config
		wantValid bool
	}{
		{
			name: "数据库连接数边界值",
			config: &Config{
				Database: DatabaseConfig{
					MaxOpenConns: 1,
				},
				Server: ServerConfig{
					Port: "8080",
				},
			},
			wantValid: true,
		},
		{
			name: "数据库连接数过大",
			config: &Config{
				Database: DatabaseConfig{
					MaxOpenConns: 2000,
				},
				Server: ServerConfig{
					Port: "8080",
				},
			},
			wantValid: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.Validate(tt.config)
			if result.IsValid != tt.wantValid {
				t.Errorf("期望验证结果为 %v，实际为 %v", tt.wantValid, result.IsValid)
				for i, err := range result.Errors {
					t.Logf("错误 %d: Field=%s, Message=%s", i, err.Field, err.Message)
				}
			}
		})
	}
}