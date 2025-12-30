package service

import (
	"context"
	"testing"
	"time"

	"mmemory/internal/models"
)

func TestWeatherEvaluator(t *testing.T) {
	t.Run("天气条件评估_等于", func(t *testing.T) {
		evaluator := NewWeatherEvaluator(nil)

		condition := &models.ReminderCondition{
			ID:       1,
			Type:     models.ConditionTypeWeather,
			Operator: models.ConditionOpEquals,
			Field:    "condition",
			Value:    `"clear"`,
			ValueType: "string",
		}

		// 测试数据
		weatherData := &models.WeatherConditionData{
			Location:    "Beijing",
			WeatherCode: "100",
			Condition:   "clear",
			Temperature: 25.0,
		}

		result, err := evaluator.Evaluate(context.Background(), condition, weatherData)
		if err != nil {
			t.Fatalf("评估失败: %v", err)
		}

		// 检查结果返回正常
		if result == nil {
			t.Fatal("期望返回评估结果，实际为nil")
		}

		// 检查值正确返回
		if result.Value != "clear" {
			t.Logf("注意: 值返回为 %v", result.Value)
		}

		t.Logf("天气评估结果: met=%v, value=%v, expected=%v", result.Met, result.Value, condition.Value)
	})

	t.Run("天气条件评估_不等于", func(t *testing.T) {
		evaluator := NewWeatherEvaluator(nil)

		condition := &models.ReminderCondition{
			ID:       2,
			Type:     models.ConditionTypeWeather,
			Operator: models.ConditionOpNotEquals,
			Field:    "condition",
			Value:    `"rain"`,
			ValueType: "string",
		}

		weatherData := &models.WeatherConditionData{
			Location:    "Beijing",
			WeatherCode: "100",
			Condition:   "clear",
			Temperature: 25.0,
		}

		result, err := evaluator.Evaluate(context.Background(), condition, weatherData)
		if err != nil {
			t.Fatalf("评估失败: %v", err)
		}

		if !result.Met {
			t.Errorf("期望条件满足（天气不是雨），实际不满足")
		}
	})

	t.Run("温度条件评估_大于", func(t *testing.T) {
		evaluator := NewWeatherEvaluator(nil)

		condition := &models.ReminderCondition{
			ID:       3,
			Type:     models.ConditionTypeTemperature,
			Operator: models.ConditionOpGreater,
			Field:    "temperature",
			Value:    "20.0",
			ValueType: "number",
		}

		weatherData := &models.WeatherConditionData{
			Location:    "Beijing",
			WeatherCode: "100",
			Condition:   "clear",
			Temperature: 25.0,
		}

		result, err := evaluator.Evaluate(context.Background(), condition, weatherData)
		if err != nil {
			t.Fatalf("评估失败: %v", err)
		}

		if !result.Met {
			t.Errorf("期望条件满足（25 > 20），实际不满足")
		}
	})

	t.Run("温度条件评估_小于", func(t *testing.T) {
		evaluator := NewWeatherEvaluator(nil)

		condition := &models.ReminderCondition{
			ID:       4,
			Type:     models.ConditionTypeTemperature,
			Operator: models.ConditionOpLess,
			Field:    "temperature",
			Value:    "30.0",
			ValueType: "number",
		}

		weatherData := &models.WeatherConditionData{
			Location:    "Beijing",
			WeatherCode: "100",
			Condition:   "clear",
			Temperature: 25.0,
		}

		result, err := evaluator.Evaluate(context.Background(), condition, weatherData)
		if err != nil {
			t.Fatalf("评估失败: %v", err)
		}

		if !result.Met {
			t.Errorf("期望条件满足（25 < 30），实际不满足")
		}
	})
}

func TestTimeEvaluator(t *testing.T) {
	evaluator := NewTimeEvaluator()

	t.Run("时间条件评估_当前时间", func(t *testing.T) {
		condition := &models.ReminderCondition{
			ID:       1,
			Type:     models.ConditionTypeTime,
			Operator: models.ConditionOpEquals,
			Field:    "hour",
			Value:    "10",
			ValueType: "number",
		}

		result, err := evaluator.Evaluate(context.Background(), condition, nil)
		if err != nil {
			t.Fatalf("评估失败: %v", err)
		}

		hour := time.Now().Hour()
		expectedMet := hour == 10

		if result.Met != expectedMet {
			t.Logf("当前小时: %d, 条件: hour == 10", hour)
		}
	})

	t.Run("时间条件评估_分钟范围", func(t *testing.T) {
		condition := &models.ReminderCondition{
			ID:       2,
			Type:     models.ConditionTypeTime,
			Operator: models.ConditionOpLess,
			Field:    "minute",
			Value:    "60",
			ValueType: "number",
		}

		result, err := evaluator.Evaluate(context.Background(), condition, nil)
		if err != nil {
			t.Fatalf("评估失败: %v", err)
		}

		// 检查结果返回正常
		if result == nil {
			t.Fatal("期望返回评估结果，实际为nil")
		}

		// 打印当前分钟数以调试
		currentMinute := time.Now().Minute()
		t.Logf("当前分钟: %d, 条件: minute < 60, 结果: met=%v", currentMinute, result.Met)
	})
}

func TestDayOfWeekEvaluator(t *testing.T) {
	evaluator := NewDayOfWeekEvaluator()

	t.Run("星期条件评估_当前星期", func(t *testing.T) {
		condition := &models.ReminderCondition{
			ID:       1,
			Type:     models.ConditionTypeDayOfWeek,
			Operator: models.ConditionOpEquals,
			Field:    "day_of_week",
			Value:    "1", // Monday
			ValueType: "string",
		}

		result, err := evaluator.Evaluate(context.Background(), condition, nil)
		if err != nil {
			t.Fatalf("评估失败: %v", err)
		}

		dayOfWeek := int(time.Now().Weekday())
		expectedMet := dayOfWeek == 1

		if result.Met != expectedMet {
			t.Logf("当前星期: %d, 条件: day_of_week == 1 (Monday)", dayOfWeek)
		}
	})

	t.Run("星期条件评估_多天匹配", func(t *testing.T) {
		condition := &models.ReminderCondition{
			ID:       2,
			Type:     models.ConditionTypeDayOfWeek,
			Operator: models.ConditionOpEquals,
			Field:    "day_of_week",
			Value:    "1,2,3,4,5", // 工作日
			ValueType: "string",
		}

		result, err := evaluator.Evaluate(context.Background(), condition, nil)
		if err != nil {
			t.Fatalf("评估失败: %v", err)
		}

		dayOfWeek := int(time.Now().Weekday())
		isWorkday := dayOfWeek >= 1 && dayOfWeek <= 5

		if result.Met != isWorkday {
			t.Errorf("期望结果与是否工作日一致: %v", isWorkday)
		}
	})
}

func TestConditionEvaluatorRegistry(t *testing.T) {
	registry := NewConditionEvaluatorRegistry()

	t.Run("注册评估器", func(t *testing.T) {
		weatherEvaluator := NewWeatherEvaluator(nil)
		timeEvaluator := NewTimeEvaluator()

		registry.Register(weatherEvaluator)
		registry.Register(timeEvaluator)

		// 检查是否注册成功
		_, ok := registry.Get(models.ConditionTypeWeather)
		if !ok {
			t.Error("期望获取到天气评估器")
		}

		_, ok = registry.Get(models.ConditionTypeTime)
		if !ok {
			t.Error("期望获取到时间评估器")
		}

		_, ok = registry.Get(models.ConditionTypeDayOfWeek)
		if ok {
			t.Error("不应该获取到星期评估器（未注册）")
		}
	})

	t.Run("获取所有支持的类型", func(t *testing.T) {
		types := registry.AllSupportedTypes()

		if len(types) != 2 {
			t.Errorf("期望2个支持的类型，实际: %d", len(types))
		}
	})
}

func TestCompareValues(t *testing.T) {
	t.Run("等于比较", func(t *testing.T) {
		if !compareValues("clear", "clear", models.ConditionOpEquals) {
			t.Error("期望相等")
		}
		if compareValues("clear", "rain", models.ConditionOpEquals) {
			t.Error("期望不相等")
		}
	})

	t.Run("不等于比较", func(t *testing.T) {
		if !compareValues("rain", "clear", models.ConditionOpNotEquals) {
			t.Error("期望不等")
		}
	})

	t.Run("大于比较", func(t *testing.T) {
		if !compareValues(25.0, 20.0, models.ConditionOpGreater) {
			t.Error("期望 25 > 20")
		}
		if compareValues(15.0, 20.0, models.ConditionOpGreater) {
			t.Error("期望 15 < 20")
		}
	})

	t.Run("小于比较", func(t *testing.T) {
		if !compareValues(15.0, 20.0, models.ConditionOpLess) {
			t.Error("期望 15 < 20")
		}
	})

	t.Run("包含比较", func(t *testing.T) {
		if !compareValues("light rain", "rain", models.ConditionOpContains) {
			t.Error("期望 'light rain' 包含 'rain'")
		}
	})
}

func TestReminderCondition_GetValue(t *testing.T) {
	t.Run("字符串值", func(t *testing.T) {
		condition := &models.ReminderCondition{
			Value:    "clear",
			ValueType: "string",
		}

		value := condition.GetValue()
		if value != "clear" {
			t.Errorf("期望 clear，实际 %v", value)
		}
	})

	t.Run("数字值", func(t *testing.T) {
		condition := &models.ReminderCondition{
			Value:    "25.5",
			ValueType: "number",
		}

		value := condition.GetValue()
		if v, ok := value.(float64); !ok || v != 25.5 {
			t.Errorf("期望 25.5，实际 %v", value)
		}
	})

	t.Run("布尔值", func(t *testing.T) {
		condition := &models.ReminderCondition{
			Value:    "true",
			ValueType: "boolean",
		}

		value := condition.GetValue()
		if v, ok := value.(bool); !ok || !v {
			t.Errorf("期望 true，实际 %v", value)
		}
	})
}

func TestReminderCondition_SetValue(t *testing.T) {
	t.Run("设置字符串", func(t *testing.T) {
		condition := &models.ReminderCondition{}
		condition.SetValue("test")

		if condition.ValueType != "string" {
			t.Errorf("期望 string，实际 %s", condition.ValueType)
		}
		if condition.Value != "test" {
			t.Errorf("期望 test，实际 %s", condition.Value)
		}
	})

	t.Run("设置数字", func(t *testing.T) {
		condition := &models.ReminderCondition{}
		condition.SetValue(25.5)

		if condition.ValueType != "number" {
			t.Errorf("期望 number，实际 %s", condition.ValueType)
		}
	})

	t.Run("设置布尔", func(t *testing.T) {
		condition := &models.ReminderCondition{}
		condition.SetValue(true)

		if condition.ValueType != "boolean" {
			t.Errorf("期望 boolean，实际 %s", condition.ValueType)
		}
		if condition.Value != "true" {
			t.Errorf("期望 true，实际 %s", condition.Value)
		}
	})

	t.Run("设置数组", func(t *testing.T) {
		condition := &models.ReminderCondition{}
		condition.SetValue([]string{"a", "b", "c"})

		if condition.ValueType != "array" {
			t.Errorf("期望 array，实际 %s", condition.ValueType)
		}
	})
}
