package service

import (
	"context"
	"testing"
	"time"

	"mmemory/internal/models"
	"mmemory/pkg/config"
	"github.com/stretchr/testify/assert"
)

// TestQWeatherService_GetCurrentWeather 测试获取当前天气
func TestQWeatherService_GetCurrentWeather(t *testing.T) {
	// 准备测试配置
	weatherConfig := &config.WeatherConfig{
		Enabled: true,
		Provider: config.WeatherProvider{
			Provider: "qweather",
			QWeather: config.QWeatherConfig{
				APIKey:  "test_api_key",
				BaseURL: "https://devapi.qweather.com/v7",
				Timeout: 5 * time.Second,
			},
		},
		Timeout:   10 * time.Second,
		MaxRetries: 3,
	}

	// 创建天气服务
	service := NewQWeatherService(weatherConfig)

	// 验证服务创建成功
	assert.NotNil(t, service, "天气服务应创建成功")
	assert.IsType(t, &QWeatherService{}, service, "应为 QWeatherService 类型")
}

// TestWeatherCondition_Check 测试天气条件检查
func TestWeatherCondition_Check(t *testing.T) {
	tests := []struct {
		name      string
		condition models.WeatherCondition
		weatherCode string
		expected   bool
	}{
		{
			name: "晴天条件 - 匹配晴天",
			condition: models.WeatherCondition{
				Condition: models.WeatherConditionClear,
				Operator:  models.OperatorEquals,
			},
			weatherCode: "100", // 晴天
			expected:    true,
		},
		{
			name: "晴天条件 - 不匹配雨天",
			condition: models.WeatherCondition{
				Condition: models.WeatherConditionClear,
				Operator:  models.OperatorEquals,
			},
			weatherCode: "305", // 小雨
			expected:    false,
		},
		{
			name: "雨天条件 - 匹配雨天",
			condition: models.WeatherCondition{
				Condition: models.WeatherConditionLightRain,
				Operator:  models.OperatorEquals,
			},
			weatherCode: "305", // 小雨
			expected:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.condition.Check(tt.weatherCode)
			assert.Equal(t, tt.expected, result, "天气条件检查结果应匹配预期")
		})
	}
}

// TestMapWeatherCodeToCondition 测试天气代码映射
func TestMapWeatherCodeToCondition(t *testing.T) {
	tests := []struct {
		name       string
		weatherCode string
		expected   models.WeatherConditionType
	}{
		{"晴天", "100", models.WeatherConditionClear},
		{"多云", "101", models.WeatherConditionClear},
		{"小雨", "305", models.WeatherConditionLightRain},
		{"中雨", "306", models.WeatherConditionRain},
		{"大雨", "307", models.WeatherConditionHeavyRain},
		{"小雪", "400", models.WeatherConditionSnow},
		{"雾", "509", models.WeatherConditionFog},
		{"未知代码", "999", models.WeatherConditionClear}, // 默认为晴天
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := models.MapWeatherCodeToCondition(tt.weatherCode)
			assert.Equal(t, tt.expected, result, "天气代码映射结果应匹配预期")
		})
	}
}

// TestWeatherConditionChecker_CheckCondition 测试天气条件检查器
func TestWeatherConditionChecker_CheckCondition(t *testing.T) {
	// 创建模拟天气服务
	mockWeatherService := &MockWeatherService{
		weather: &models.NowWeather{
			WeatherCode: "305",
			WeatherDesc: "小雨",
		},
	}

	checker := NewWeatherConditionChecker(mockWeatherService)
	assert.NotNil(t, checker, "天气条件检查器应创建成功")

	// 测试条件匹配
	condition := &models.WeatherCondition{
		ID:         1,
		Location:   "北京",
		Condition:  models.WeatherConditionLightRain,
		Operator:   models.OperatorEquals,
	}

	ctx := context.Background()
	result, err := checker.CheckCondition(ctx, condition)
	assert.NoError(t, err, "检查条件不应返回错误")
	assert.True(t, result, "小雨条件应匹配天气代码305")
}

// MockWeatherService 模拟天气服务
type MockWeatherService struct {
	weather *models.NowWeather
	err     error
}

func (m *MockWeatherService) GetCurrentWeather(ctx context.Context, location string) (*models.NowWeather, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.weather, nil
}

func (m *MockWeatherService) GetWeatherCondition(ctx context.Context, location string) (*models.WeatherCondition, error) {
	if m.err != nil {
		return nil, m.err
	}
	weather := m.weather
	if weather == nil {
		return nil, nil
	}

	return &models.WeatherCondition{
		Condition: models.MapWeatherCodeToCondition(weather.WeatherCode),
		Location:  location,
		Operator:  models.OperatorEquals,
	}, nil
}

// TestWeatherResponse_IsSuccess 测试天气响应状态检查
func TestWeatherResponse_IsSuccess(t *testing.T) {
	tests := []struct {
		name   string
		code   string
		expect bool
	}{
		{"成功响应", "200", true},
		{"失败响应", "400", false},
		{"其他错误", "404", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &models.WeatherResponse{Code: tt.code}
			assert.Equal(t, tt.expect, resp.IsSuccess(), "响应状态检查应匹配预期")
		})
	}
}

// TestWeatherResponse_GetCurrentWeatherCondition 测试获取当前天气条件
func TestWeatherResponse_GetCurrentWeatherCondition(t *testing.T) {
	resp := &models.WeatherResponse{
		Now: models.NowWeather{
			WeatherCode: "305", // 小雨
		},
	}

	condition := resp.GetCurrentWeatherCondition()
	assert.Equal(t, models.WeatherConditionLightRain, condition, "应正确映射天气代码到条件类型")
}
