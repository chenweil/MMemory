package models

import "time"

// WeatherConditionType 天气条件类型
type WeatherConditionType string

const (
	WeatherConditionClear     WeatherConditionType = "clear"     // 晴天
	WeatherConditionCloudy    WeatherConditionType = "cloudy"    // 多云
	WeatherConditionOvercast  WeatherConditionType = "overcast"  // 阴天
	WeatherConditionLightRain WeatherConditionType = "light_rain" // 小雨
	WeatherConditionRain      WeatherConditionType = "rain"      // 中雨
	WeatherConditionHeavyRain WeatherConditionType = "heavy_rain" // 大雨
	WeatherConditionSnow      WeatherConditionType = "snow"      // 雪
	WeatherConditionFog       WeatherConditionType = "fog"       // 雾
)

// WeatherConditionOperator 天气条件操作符
type WeatherConditionOperator string

const (
	OperatorEquals    WeatherConditionOperator = "equals"    // 等于
	OperatorNotEquals WeatherConditionOperator = "not_equals" // 不等于
	OperatorContains  WeatherConditionOperator = "contains"  // 包含
	OperatorStartsWith WeatherConditionOperator = "starts_with" // 以...开头
)

// WeatherCondition 天气条件模型（用于条件提醒）
type WeatherCondition struct {
	ID          uint                  `gorm:"primaryKey;autoIncrement" json:"id"`
	ReminderID  uint                  `gorm:"not null;index" json:"reminder_id"`
	Location    string                `gorm:"size:100;not null" json:"location"`
	Condition   WeatherConditionType  `gorm:"size:20;not null" json:"condition"`
	Operator    WeatherConditionOperator `gorm:"size:20;not null" json:"operator"`
	Description string                `gorm:"size:200" json:"description"`
	CreatedAt   time.Time             `json:"created_at"`
	UpdatedAt   time.Time             `json:"updated_at"`

	// 关联关系
	Reminder Reminder `gorm:"foreignKey:ReminderID" json:"reminder,omitempty"`
}

// TableName 指定表名
func (WeatherCondition) TableName() string {
	return "weather_conditions"
}

// Check 验证天气条件
func (wc *WeatherCondition) Check(weatherCode string) bool {
	// 将和风天气代码映射为条件类型
	conditionType := MapWeatherCodeToCondition(weatherCode)

	switch wc.Operator {
	case OperatorEquals:
		return conditionType == wc.Condition
	case OperatorNotEquals:
		return conditionType != wc.Condition
	case OperatorContains, OperatorStartsWith:
		// 对于中文描述，支持包含操作
		return false // 暂时不实现
	default:
		return false
	}
}

// MapWeatherCodeToCondition 将和风天气代码映射为条件类型
func MapWeatherCodeToCondition(code string) WeatherConditionType {
	// 和风天气代码映射
	// 100: 晴 101: 多云 102: 多云 103: 晴间多云 104: 晴间多云
	// 300: 阵雨 301: 强阵雨 302: 雷阵雨 303: 强雷阵雨 304: 冰雹
	// 305: 小雨 306: 中雨 307: 大雨 308: 极端降雨 309: 毛毛雨
	// 310: 暴雨 311: 大暴雨 312: 特大暴雨 313: 冻雨
	// 400: 小雪 401: 中雪 402: 大雪 403: 暴雪 404: 雨雪天气 405: 雨夹雪
	// 406: 阵雨夹雪 407: 阵雪
	// 500: 薄雾 501: 霾 502: 扬沙 503: 浮尘 504: 沙尘暴 507: 沙尘暴
	// 508: 强沙尘暴 509: 雾 510: 霾 511: 扬沙 512: 强沙尘暴

	switch code {
	case "100", "101", "102", "103", "104":
		return WeatherConditionClear
	case "300", "301":
		return WeatherConditionCloudy
	case "302", "303", "304":
		return WeatherConditionOvercast
	case "305", "309":
		return WeatherConditionLightRain
	case "306", "308", "310", "311", "312", "313":
		return WeatherConditionRain
	case "307":
		return WeatherConditionHeavyRain
	case "400", "401", "402", "403", "404", "405", "406", "407":
		return WeatherConditionSnow
	case "500", "509":
		return WeatherConditionFog
	default:
		return WeatherConditionClear
	}
}

// WeatherResponse 和风天气API响应
type WeatherResponse struct {
	Code       string    `json:"code"`       // 状态码
	UpdateTime string    `json:"updateTime"` // 更新时间
	FxLink     string    `json:"fxLink"`     // 链接
	Now        NowWeather `json:"now"`       // 实况天气
	// 预报数据省略
}

// NowWeather 当前天气
type NowWeather struct {
	ObsTime    string `json:"obsTime"`    // 观测时间
	Temp       string `json:"temp"`       // 温度（摄氏度）
	FeelsLike  string `json:"feelsLike"`  // 体感温度
	Icon       string `json:"icon"`       // 天气现象图标
	WeatherCode string `json:"weatherCode"` // 天气现象代码
	WeatherDesc string `json:"weatherText"` // 天气现象文字
	Wind360    string `json:"wind360"`    // 风向360角度
	WindDir    string `json:"windDir"`    // 风向
	WindScale  string `json:"windScale"`  // 风力等级
	WindSpeed  string `json:"windSpeed"`  // 风速（公里/小时）
	Humidity   string `json:"humidity"`   // 相对湿度（百分比）
	Precip     string `json:"precip"`     // 降水量（毫米）
	Pressure   string `json:"pressure"`   // 大气压强（百帕）
	Vis        string `json:"vis"`        // 能见度（公里）
	Cloud      string `json:"cloud"`      // 云量（百分比）
	Dew        string `json:"dew"`        // 露点温度
}

// LocationInfo 位置信息
type LocationInfo struct {
	Location  string  `json:"location"`  // 位置ID或城市名
	Name      string  `json:"name"`      // 城市名称
	Adm2      string  `json:"adm2"`      // 所属行政区2
	Adm1      string  `json:"adm1"`      // 所属行政区1
	Country   string  `json:"country"`   // 所属国家
	Lat       string  `json:"lat"`       // 纬度
	Lon       string  `json:"lon"`       // 经度
	Timezone  string  `json:"tz"`        // 时区
	UTC       string  `json:"utcOffset"` // UTC偏移
	IsDst     string  `json:"isDst"`     // 是否夏令时
	Type      string  `json:"type"`      // 类型
	Rank      string  `json:"rank"`      // 行政区域 Ranking
	FxLink    string  `json:"fxLink"`    // 链接
}

// GetCurrentWeatherCondition 获取当前天气条件类型
func (wr *WeatherResponse) GetCurrentWeatherCondition() WeatherConditionType {
	return MapWeatherCodeToCondition(wr.Now.WeatherCode)
}

// IsSuccess 检查响应是否成功
func (wr *WeatherResponse) IsSuccess() bool {
	return wr.Code == "200"
}
