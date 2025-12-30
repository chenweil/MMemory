package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"mmemory/internal/models"
	"mmemory/pkg/config"
	"mmemory/pkg/logger"
)

// WeatherService 天气服务接口
type WeatherService interface {
	GetCurrentWeather(ctx context.Context, location string) (*models.NowWeather, error)
	GetWeatherCondition(ctx context.Context, location string) (*models.WeatherCondition, error)
}

// QWeatherService 和风天气服务实现
type QWeatherService struct {
	config  *config.WeatherConfig
	client  *http.Client
	apiKey  string
	baseURL string
	timeout time.Duration
}

// NewQWeatherService 创建和风天气服务
func NewQWeatherService(weatherConfig *config.WeatherConfig) WeatherService {
	return &QWeatherService{
		config:  weatherConfig,
		apiKey:  weatherConfig.Provider.QWeather.APIKey,
		baseURL: weatherConfig.Provider.QWeather.BaseURL,
		timeout: weatherConfig.Provider.QWeather.Timeout,
		client: &http.Client{
			Timeout: weatherConfig.Provider.QWeather.Timeout,
		},
	}
}

// GetCurrentWeather 获取当前天气
func (ws *QWeatherService) GetCurrentWeather(ctx context.Context, location string) (*models.NowWeather, error) {
	// 构建请求URL
	url := fmt.Sprintf("%s/weather/now?location=%s&key=%s", ws.baseURL, location, ws.apiKey)

	// 创建HTTP请求
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		logger.Errorf("创建天气请求失败: %v", err)
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	// 发送请求
	resp, err := ws.client.Do(req)
	if err != nil {
		logger.Errorf("天气API请求失败: %v", err)
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		logger.Errorf("天气API响应错误: %d", resp.StatusCode)
		return nil, fmt.Errorf("API响应错误: %d", resp.StatusCode)
	}

	// 解析响应
	var weatherResp models.WeatherResponse
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&weatherResp); err != nil {
		logger.Errorf("解析天气响应失败: %v", err)
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	// 检查API响应状态
	if !weatherResp.IsSuccess() {
		logger.Errorf("天气API返回错误: %s", weatherResp.Code)
		return nil, fmt.Errorf("API返回错误: %s", weatherResp.Code)
	}

	return &weatherResp.Now, nil
}

// GetWeatherCondition 获取天气条件
func (ws *QWeatherService) GetWeatherCondition(ctx context.Context, location string) (*models.WeatherCondition, error) {
	now, err := ws.GetCurrentWeather(ctx, location)
	if err != nil {
		return nil, err
	}

	// 转换为天气条件
	conditionType := models.MapWeatherCodeToCondition(now.WeatherCode)

	return &models.WeatherCondition{
		Condition: conditionType,
		Location:  location,
		Operator:  models.OperatorEquals,
	}, nil
}

// WeatherConditionChecker 天气条件检查器
type WeatherConditionChecker struct {
	weatherService WeatherService
}

// NewWeatherConditionChecker 创建天气条件检查器
func NewWeatherConditionChecker(weatherService WeatherService) *WeatherConditionChecker {
	return &WeatherConditionChecker{
		weatherService: weatherService,
	}
}

// CheckCondition 检查天气条件是否满足
func (wcc *WeatherConditionChecker) CheckCondition(ctx context.Context, condition *models.WeatherCondition) (bool, error) {
	// 获取当前天气
	nowWeather, err := wcc.weatherService.GetCurrentWeather(ctx, condition.Location)
	if err != nil {
		logger.Errorf("获取天气信息失败: %v", err)
		return false, fmt.Errorf("获取天气信息失败: %w", err)
	}

	// 检查条件
	return condition.Check(nowWeather.WeatherCode), nil
}

// BatchCheckConditions 批量检查多个天气条件
func (wcc *WeatherConditionChecker) BatchCheckConditions(ctx context.Context, conditions []*models.WeatherCondition) (map[uint]bool, error) {
	results := make(map[uint]bool)
	locationMap := make(map[string][]*models.WeatherCondition)

	// 按位置分组
	for _, condition := range conditions {
		locationMap[condition.Location] = append(locationMap[condition.Location], condition)
	}

	// 获取每个位置的天气信息
	for location, locationConditions := range locationMap {
		nowWeather, err := wcc.weatherService.GetCurrentWeather(ctx, location)
		if err != nil {
			logger.Errorf("获取位置 %s 天气信息失败: %v", location, err)
			// 继续处理其他位置
			continue
		}

		// 检查每个条件
		for _, condition := range locationConditions {
			results[condition.ID] = condition.Check(nowWeather.WeatherCode)
		}
	}

	return results, nil
}
