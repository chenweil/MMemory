package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"mmemory/internal/models"
	"mmemory/pkg/logger"
)

// ConditionEvaluator 条件评估器接口
type ConditionEvaluator interface {
	Evaluate(ctx context.Context, condition *models.ReminderCondition, data interface{}) (*models.ConditionEvaluationResult, error)
	GetConditionType() models.ConditionType
}

// WeatherEvaluator 天气条件评估器
type WeatherEvaluator struct {
	weatherService WeatherService
}

// NewWeatherEvaluator 创建天气条件评估器
func NewWeatherEvaluator(weatherService WeatherService) *WeatherEvaluator {
	return &WeatherEvaluator{
		weatherService: weatherService,
	}
}

// GetConditionType 获取支持的條件類型
func (we *WeatherEvaluator) GetConditionType() models.ConditionType {
	return models.ConditionTypeWeather
}

// Evaluate 评估天气条件
func (we *WeatherEvaluator) Evaluate(ctx context.Context, condition *models.ReminderCondition, data interface{}) (*models.ConditionEvaluationResult, error) {
	result := &models.ConditionEvaluationResult{
		ConditionID: condition.ID,
		Type:        string(models.ConditionTypeWeather),
		Operator:    string(condition.Operator),
		CheckedAt:   time.Now(),
	}

	// 获取天气数据
	var weatherData *models.WeatherConditionData
	if wd, ok := data.(*models.WeatherConditionData); ok {
		weatherData = wd
	} else if location, ok := data.(string); ok {
		// 如果传入的是位置字符串，获取天气
		wd, err := we.getWeatherData(ctx, location)
		if err != nil {
			result.Details = fmt.Sprintf("获取天气数据失败: %v", err)
			return result, err
		}
		weatherData = wd
	} else {
		return nil, fmt.Errorf("无效的天气数据")
	}

	// 设置值和期望值
	result.Value = weatherData.Condition
	result.Expected = condition.GetValue()

	// 评估条件
	result.Met = we.evaluateCondition(weatherData, condition)
	result.Description = fmt.Sprintf("天气 %s %s %s", weatherData.Condition, condition.Operator, condition.Value)

	return result, nil
}

// getWeatherData 获取天气数据
func (we *WeatherEvaluator) getWeatherData(ctx context.Context, location string) (*models.WeatherConditionData, error) {
	now, err := we.weatherService.GetCurrentWeather(ctx, location)
	if err != nil {
		return nil, err
	}

	temp, _ := strconv.ParseFloat(now.Temp, 64)
	humidity, _ := strconv.ParseFloat(now.Humidity, 64)
	windSpeed, _ := strconv.ParseFloat(now.WindSpeed, 64)
	precip, _ := strconv.ParseFloat(now.Precip, 64)
	cloud, _ := strconv.ParseFloat(now.Cloud, 64)
	vis, _ := strconv.ParseFloat(now.Vis, 64)

	return &models.WeatherConditionData{
		Location:    location,
		WeatherCode: now.WeatherCode,
		Temperature: temp,
		Humidity:    humidity,
		WindSpeed:   windSpeed,
		Condition:   string(models.MapWeatherCodeToCondition(now.WeatherCode)),
		Precip:      precip,
		CloudCover:  cloud,
		Visibility:  vis,
		FetchedAt:   time.Now(),
	}, nil
}

// evaluateCondition 评估具体条件
func (we *WeatherEvaluator) evaluateCondition(data *models.WeatherConditionData, condition *models.ReminderCondition) bool {
	field := condition.Field
	if field == "" {
		field = "condition" // 默认检查天气状况
	}

	expected := condition.GetValue()
	actual := we.getFieldValue(data, field)

	if actual == nil {
		return false
	}

	result := compareValues(actual, expected, condition.Operator)

	if condition.IsNegated {
		return !result
	}
	return result
}

// getFieldValue 获取字段值
func (we *WeatherEvaluator) getFieldValue(data *models.WeatherConditionData, field string) interface{} {
	switch field {
	case "condition":
		return data.Condition
	case "temperature":
		return data.Temperature
	case "humidity":
		return data.Humidity
	case "wind_speed":
		return data.WindSpeed
	case "precip":
		return data.Precip
	case "cloud_cover":
		return data.CloudCover
	case "visibility":
		return data.Visibility
	case "weather_code":
		return data.WeatherCode
	}
	return nil
}

// TimeEvaluator 时间条件评估器
type TimeEvaluator struct{}

// NewTimeEvaluator 创建时间条件评估器
func NewTimeEvaluator() *TimeEvaluator {
	return &TimeEvaluator{}
}

// GetConditionType 获取支持的條件類型
func (te *TimeEvaluator) GetConditionType() models.ConditionType {
	return models.ConditionTypeTime
}

// Evaluate 评估时间条件
func (te *TimeEvaluator) Evaluate(ctx context.Context, condition *models.ReminderCondition, data interface{}) (*models.ConditionEvaluationResult, error) {
	result := &models.ConditionEvaluationResult{
		ConditionID: condition.ID,
		Type:        string(models.ConditionTypeTime),
		Operator:    string(condition.Operator),
		CheckedAt:   time.Now(),
	}

	now := time.Now()
	timeData := &models.TimeConditionData{
		CurrentTime: now,
		Hour:        now.Hour(),
		Minute:      now.Minute(),
		DayOfWeek:   int(now.Weekday()),
		DayOfMonth:  now.Day(),
		Month:       int(now.Month()),
		Date:        now.Format("2006-01-02"),
		Time:        now.Format("15:04:05"),
	}

	// 设置值和期望值
	result.Value = timeData.CurrentTime.Format("15:04:05")
	result.Expected = condition.GetValue()

	result.Met = te.evaluateCondition(timeData, condition)
	result.Description = fmt.Sprintf("时间 %s %s %s", timeData.Time, condition.Operator, condition.Value)

	return result, nil
}

// evaluateCondition 评估时间条件
func (te *TimeEvaluator) evaluateCondition(data *models.TimeConditionData, condition *models.ReminderCondition) bool {
	field := condition.Field
	if field == "" {
		field = "time" // 默认检查时间
	}

	expected := condition.GetValue()
	actual := te.getFieldValue(data, field)

	if actual == nil {
		return false
	}

	result := compareValues(actual, expected, condition.Operator)

	if condition.IsNegated {
		return !result
	}
	return result
}

// getFieldValue 获取字段值
func (te *TimeEvaluator) getFieldValue(data *models.TimeConditionData, field string) interface{} {
	switch field {
	case "time":
		return data.Time
	case "hour":
		return data.Hour
	case "minute":
		return data.Minute
	case "day_of_week":
		return data.DayOfWeek
	case "day_of_month":
		return data.DayOfMonth
	case "month":
		return data.Month
	case "date":
		return data.Date
	}
	return nil
}

// DayOfWeekEvaluator 星期条件评估器
type DayOfWeekEvaluator struct{}

// NewDayOfWeekEvaluator 创建星期条件评估器
func NewDayOfWeekEvaluator() *DayOfWeekEvaluator {
	return &DayOfWeekEvaluator{}
}

// GetConditionType 获取支持的條件類型
func (de *DayOfWeekEvaluator) GetConditionType() models.ConditionType {
	return models.ConditionTypeDayOfWeek
}

// Evaluate 评估星期条件
func (de *DayOfWeekEvaluator) Evaluate(ctx context.Context, condition *models.ReminderCondition, data interface{}) (*models.ConditionEvaluationResult, error) {
	result := &models.ConditionEvaluationResult{
		ConditionID: condition.ID,
		Type:        string(models.ConditionTypeDayOfWeek),
		Operator:    string(condition.Operator),
		CheckedAt:   time.Now(),
	}

	now := time.Now()
	dayOfWeek := int(now.Weekday())

	result.Value = dayOfWeek
	result.Expected = condition.GetValue()

	// 解析期望值（可能是逗号分隔的列表）
	expectedDays := parseDayOfWeekValues(condition.Value)
	result.Met = containsDay(expectedDays, dayOfWeek)

	if condition.IsNegated {
		result.Met = !result.Met
	}

	result.Description = fmt.Sprintf("星期 %d %s %s", dayOfWeek, condition.Operator, condition.Value)

	return result, nil
}

// parseDayOfWeekValues 解析星期值
func parseDayOfWeekValues(value string) []int {
	var days []int
	parts := strings.Split(value, ",")
	for _, part := range parts {
		day, err := strconv.Atoi(strings.TrimSpace(part))
		if err == nil && day >= 0 && day <= 6 {
			days = append(days, day)
		}
	}
	return days
}

// containsDay 检查是否包含某天
func containsDay(days []int, day int) bool {
	for _, d := range days {
		if d == day {
			return true
		}
	}
	return false
}

// TemperatureEvaluator 温度条件评估器
type TemperatureEvaluator struct {
	weatherService WeatherService
}

// NewTemperatureEvaluator 创建温度条件评估器
func NewTemperatureEvaluator(weatherService WeatherService) *TemperatureEvaluator {
	return &TemperatureEvaluator{
		weatherService: weatherService,
	}
}

// GetConditionType 获取支持的條件類型
func (te *TemperatureEvaluator) GetConditionType() models.ConditionType {
	return models.ConditionTypeTemperature
}

// Evaluate 评估温度条件
func (te *TemperatureEvaluator) Evaluate(ctx context.Context, condition *models.ReminderCondition, data interface{}) (*models.ConditionEvaluationResult, error) {
	result := &models.ConditionEvaluationResult{
		ConditionID: condition.ID,
		Type:        string(models.ConditionTypeTemperature),
		Operator:    string(condition.Operator),
		CheckedAt:   time.Now(),
	}

	var temp float64

	if td, ok := data.(*models.WeatherConditionData); ok {
		temp = td.Temperature
	} else if location, ok := data.(string); ok {
		// 需要获取天气数据
		wd, err := te.weatherService.GetCurrentWeather(ctx, location)
		if err != nil {
			return nil, err
		}
		temp, _ = strconv.ParseFloat(wd.Temp, 64)
	} else {
		return nil, fmt.Errorf("无效的温度数据")
	}

	result.Value = temp
	result.Expected = condition.GetValue()
	result.Met = compareValues(temp, condition.GetValue(), condition.Operator)

	if condition.IsNegated {
		result.Met = !result.Met
	}

	result.Description = fmt.Sprintf("温度 %.1f°C %s %v", temp, condition.Operator, condition.GetValue())

	return result, nil
}

// ConditionEvaluatorRegistry 条件评估器注册表
type ConditionEvaluatorRegistry struct {
	evaluators map[models.ConditionType]ConditionEvaluator
	mu         sync.RWMutex
}

// NewConditionEvaluatorRegistry 创建评估器注册表
func NewConditionEvaluatorRegistry() *ConditionEvaluatorRegistry {
	return &ConditionEvaluatorRegistry{
		evaluators: make(map[models.ConditionType]ConditionEvaluator),
	}
}

// Register 注册评估器
func (cer *ConditionEvaluatorRegistry) Register(evaluator ConditionEvaluator) {
	cer.mu.Lock()
	defer cer.mu.Unlock()
	cer.evaluators[evaluator.GetConditionType()] = evaluator
}

// Get 获取评估器
func (cer *ConditionEvaluatorRegistry) Get(conditionType models.ConditionType) (ConditionEvaluator, bool) {
	cer.mu.RLock()
	defer cer.mu.RUnlock()
	evaluator, ok := cer.evaluators[conditionType]
	return evaluator, ok
}

// AllSupportedTypes 获取所有支持的类型
func (cer *ConditionEvaluatorRegistry) AllSupportedTypes() []models.ConditionType {
	cer.mu.RLock()
	defer cer.mu.RUnlock()
	types := make([]models.ConditionType, 0, len(cer.evaluators))
	for t := range cer.evaluators {
		types = append(types, t)
	}
	return types
}

// compareValues 比较值
func compareValues(actual, expected interface{}, operator models.ConditionOperator) bool {
	switch op := operator; op {
	case models.ConditionOpEquals:
		return fmt.Sprintf("%v", actual) == fmt.Sprintf("%v", expected)
	case models.ConditionOpNotEquals:
		return fmt.Sprintf("%v", actual) != fmt.Sprintf("%v", expected)
	case models.ConditionOpGreater:
		if a, ok := actual.(float64); ok {
			if e, ok := expected.(float64); ok {
				return a > e
			}
		}
	case models.ConditionOpLess:
		if a, ok := actual.(float64); ok {
			if e, ok := expected.(float64); ok {
				return a < e
			}
		}
	case models.ConditionOpContains:
		if a, ok := actual.(string); ok {
			if e, ok := expected.(string); ok {
				return strings.Contains(a, e)
			}
		}
	case models.ConditionOpIn:
		// 检查actual是否在expected列表中
		if arr, ok := expected.([]string); ok {
			aStr := fmt.Sprintf("%v", actual)
			for _, v := range arr {
				if aStr == v {
					return true
				}
			}
		}
	}
	return false
}

// ConditionService 条件服务
type ConditionService struct {
	registry      *ConditionEvaluatorRegistry
	weatherSvc    WeatherService
	reminderSvc   ReminderService
	conditionRepo interface {
		GetByReminderID(ctx context.Context, reminderID uint) ([]models.ReminderCondition, error)
		Create(ctx context.Context, condition *models.ReminderCondition) error
		Delete(ctx context.Context, id uint) error
	}
}

// NewConditionService 创建条件服务
func NewConditionService(
	weatherSvc WeatherService,
	reminderSvc ReminderService,
) *ConditionService {
	svc := &ConditionService{
		registry:    NewConditionEvaluatorRegistry(),
		weatherSvc:  weatherSvc,
		reminderSvc: reminderSvc,
	}

	// 注册内置评估器
	svc.registry.Register(NewWeatherEvaluator(weatherSvc))
	svc.registry.Register(NewTimeEvaluator())
	svc.registry.Register(NewDayOfWeekEvaluator())
	svc.registry.Register(NewTemperatureEvaluator(weatherSvc))

	return svc
}

// EvaluateConditions 评估提醒的所有条件
func (cs *ConditionService) EvaluateConditions(ctx context.Context, reminderID uint) (*models.ConditionEvaluationStatus, error) {
	// 获取提醒的条件
	conditions, err := cs.conditionRepo.GetByReminderID(ctx, reminderID)
	if err != nil {
		logger.Errorf("获取提醒条件失败: %v", err)
		return nil, err
	}

	status := &models.ConditionEvaluationStatus{
		ReminderID:    reminderID,
		Results:       make([]models.ConditionEvaluationResult, 0),
		LastCheckedAt: time.Now(),
	}

	allMet := true
	var failedConditions []string

	for _, condition := range conditions {
		result, err := cs.EvaluateCondition(ctx, &condition, nil)
		if err != nil {
			logger.Errorf("评估条件失败: %v", err)
			continue
		}

		status.Results = append(status.Results, *result)
		if !result.Met {
			allMet = false
			failedConditions = append(failedConditions, result.Description)
		}
	}

	status.AllConditionsMet = allMet
	status.FailedConditions = failedConditions

	return status, nil
}

// EvaluateCondition 评估单个条件
func (cs *ConditionService) EvaluateCondition(ctx context.Context, condition *models.ReminderCondition, data interface{}) (*models.ConditionEvaluationResult, error) {
	evaluator, ok := cs.registry.Get(condition.Type)
	if !ok {
		return nil, fmt.Errorf("不支持的条件类型: %s", condition.Type)
	}

	return evaluator.Evaluate(ctx, condition, data)
}

// GetSupportedTypes 获取所有支持的条件类型
func (cs *ConditionService) GetSupportedTypes() []models.ConditionType {
	return cs.registry.AllSupportedTypes()
}
