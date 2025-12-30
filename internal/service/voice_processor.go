package service

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"mmemory/internal/models"
	"mmemory/pkg/ai"
	"mmemory/pkg/logger"

	"github.com/sirupsen/logrus"
)

// VoiceMessage 语音消息
type VoiceMessage struct {
	FileID       string    `json:"file_id"`
	Duration     int       `json:"duration"` // 秒
	MimeType     string    `json:"mime_type"`
	FileSize     int       `json:"file_size"`
	Waveform     string    `json:"waveform,omitempty"`
	LanguageCode string    `json:"language_code"`
	UploadedAt   time.Time `json:"uploaded_at"`
}

// VoiceProcessingResult 语音处理结果
type VoiceProcessingResult struct {
	Text           string              `json:"text"`
	Confidence     float64             `json:"confidence"`
	Language       string              `json:"language"`
	Duration       int                 `json:"duration"`
	ParseResult    *ai.ParseResult     `json:"parse_result,omitempty"`
	ProcessingTime time.Duration       `json:"processing_time"`
}

// VoiceProcessingService 语音处理服务接口
type VoiceProcessingService interface {
	ProcessVoiceMessage(ctx context.Context, voice *VoiceMessage) (*VoiceProcessingResult, error)
	DownloadVoiceFile(ctx context.Context, fileID string) ([]byte, error)
	TranscribeAudio(ctx context.Context, audioData []byte, mimeType string) (*TranscriptionResult, error)
	ParseTranscribedText(ctx context.Context, text string, userID uint) (*ai.ParseResult, error)
}

// TranscriptionResult 转录结果
type TranscriptionResult struct {
	Text       string  `json:"text"`
	Confidence float64 `json:"confidence"`
	Language   string  `json:"language"`
	Duration   int     `json:"duration"`
}

// DefaultVoiceProcessingService 默认语音处理服务
type DefaultVoiceProcessingService struct {
	aiParserService AIParserService
	httpClient      HTTPClient
	botAPI          BotAPIClient
	log             *logrus.Logger
}

// HTTPClient HTTP客户端接口
type HTTPClient interface {
	Get(url string) (*http.Response, error)
	Post(url string, body interface{}) (*http.Response, error)
}

// BotAPIClient Telegram Bot API接口
type BotAPIClient interface {
	GetFile(fileID string) (*BotFile, error)
	DownloadFile(filePath string) ([]byte, error)
}

// BotFile Bot文件信息
type BotFile struct {
	FileID   string `json:"file_id"`
	FilePath string `json:"file_path"`
	FileSize int    `json:"file_size"`
}

// NewDefaultVoiceProcessingService 创建默认语音处理服务
func NewDefaultVoiceProcessingService(
	aiParserService AIParserService,
	botAPI BotAPIClient,
) *DefaultVoiceProcessingService {
	return &DefaultVoiceProcessingService{
		aiParserService: aiParserService,
		botAPI:          botAPI,
		log:             logger.GetLogger(),
	}
}

// ProcessVoiceMessage 处理语音消息
func (vps *DefaultVoiceProcessingService) ProcessVoiceMessage(ctx context.Context, voice *VoiceMessage) (*VoiceProcessingResult, error) {
	startTime := time.Now()

	// 1. 下载语音文件
	vps.log.Infof("开始处理语音消息: file_id=%s, duration=%ds", voice.FileID, voice.Duration)

	audioData, err := vps.DownloadVoiceFile(ctx, voice.FileID)
	if err != nil {
		return nil, fmt.Errorf("下载语音文件失败: %w", err)
	}

	// 2. 转录音频
	transcription, err := vps.TranscribeAudio(ctx, audioData, voice.MimeType)
	if err != nil {
		return nil, fmt.Errorf("转录音频失败: %w", err)
	}

	// 3. 解析转录文本
	var parseResult *ai.ParseResult
	parsedText, err := vps.ParseTranscribedText(ctx, transcription.Text, 0)
	if err != nil {
		vps.log.Warnf("解析转录文本失败: %v", err)
	} else {
		parseResult = parsedText
	}

	result := &VoiceProcessingResult{
		Text:           transcription.Text,
		Confidence:     transcription.Confidence,
		Language:       transcription.Language,
		Duration:       transcription.Duration,
		ParseResult:    parseResult,
		ProcessingTime: time.Since(startTime),
	}

	vps.log.Infof("语音处理完成: text='%s', processing_time=%v", result.Text, result.ProcessingTime)

	return result, nil
}

// DownloadVoiceFile 下载语音文件
func (vps *DefaultVoiceProcessingService) DownloadVoiceFile(ctx context.Context, fileID string) ([]byte, error) {
	// 获取文件路径
	file, err := vps.botAPI.GetFile(fileID)
	if err != nil {
		return nil, err
	}

	// 下载文件内容
	return vps.botAPI.DownloadFile(file.FilePath)
}

// TranscribeAudio 转录音频
func (vps *DefaultVoiceProcessingService) TranscribeAudio(ctx context.Context, audioData []byte, mimeType string) (*TranscriptionResult, error) {
	// 模拟转录结果
	// 在实际实现中，这里应该调用语音识别API（如Whisper、Google Speech-to-Text等）
	return &TranscriptionResult{
		Text:       "请每天早上提醒我喝水",
		Confidence: 0.95,
		Language:   "zh-CN",
		Duration:   3000, // 毫秒
	}, nil
}

// ParseTranscribedText 解析转录文本
func (vps *DefaultVoiceProcessingService) ParseTranscribedText(ctx context.Context, text string, userID uint) (*ai.ParseResult, error) {
	// 使用AI解析服务解析文本
	if vps.aiParserService != nil {
		return vps.aiParserService.ParseMessage(ctx, fmt.Sprintf("%d", userID), text)
	}

	// 返回模拟结果
	return &ai.ParseResult{
		Intent:     ai.IntentReminder,
		Confidence: 0.8,
		Reminder: &ai.ReminderInfo{
			Title: text,
			Type:  models.ReminderTypeTask,
			Time: ai.TimeInfo{
				Hour:   8,
				Minute: 0,
			},
			SchedulePattern: models.SchedulePatternDaily,
		},
		ParsedBy:    "mock",
		ProcessTime: time.Millisecond * 100,
		Timestamp:   time.Now(),
	}, nil
}

// WhisperSpeechToText 使用OpenAI Whisper进行语音转文字
type WhisperSpeechToText struct {
	apiKey  string
	baseURL string
	client  HTTPClient
}

// NewWhisperSpeechToText 创建Whisper转写服务
func NewWhisperSpeechToText(apiKey, baseURL string) *WhisperSpeechToText {
	return &WhisperSpeechToText{
		apiKey:  apiKey,
		baseURL: baseURL,
	}
}

// Transcribe 转录音频
func (w *WhisperSpeechToText) Transcribe(ctx context.Context, audioData []byte, mimeType string) (*TranscriptionResult, error) {
	// 这里应该调用OpenAI Whisper API
	// 实际实现需要：
	// 1. 将音频数据上传到OpenAI
	// 2. 调用 Whisper API 进行转录
	// 3. 返回转录结果

	// 模拟实现
	return &TranscriptionResult{
		Text:       "语音转文字结果",
		Confidence: 0.9,
		Language:   "zh-CN",
		Duration:   5000,
	}, nil
}

// GoogleSpeechToText 谷歌语音转文字
type GoogleSpeechToText struct {
	apiKey    string
	projectID string
	client    HTTPClient
}

// NewGoogleSpeechToText 创建谷歌语音转写服务
func NewGoogleSpeechToText(apiKey, projectID string) *GoogleSpeechToText {
	return &GoogleSpeechToText{
		apiKey:    apiKey,
		projectID: projectID,
	}
}

// Transcribe 转录音频
func (g *GoogleSpeechToText) Transcribe(ctx context.Context, audioData []byte, mimeType string) (*TranscriptionResult, error) {
	// 这里应该调用Google Cloud Speech-to-Text API
	// 实际实现需要：
	// 1. 配置Google Cloud项目
	// 2. 上传音频到GCS或直接发送
	// 3. 调用Speech-to-Text API
	// 4. 返回转录结果

	// 模拟实现
	return &TranscriptionResult{
		Text:       "语音转文字结果",
		Confidence: 0.88,
		Language:   "zh-CN",
		Duration:   4500,
	}, nil
}

// VoiceInputHandler 语音输入处理器
type VoiceInputHandler struct {
	voiceService VoiceProcessingService
	reminderSvc  ReminderService
	log          *logrus.Logger
}

// NewVoiceInputHandler 创建语音输入处理器
func NewVoiceInputHandler(
	voiceService VoiceProcessingService,
	reminderSvc ReminderService,
) *VoiceInputHandler {
	return &VoiceInputHandler{
		voiceService: voiceService,
		reminderSvc:  reminderSvc,
		log:          logger.GetLogger(),
	}
}

// HandleVoiceMessage 处理语音消息并创建提醒
func (vih *VoiceInputHandler) HandleVoiceMessage(ctx context.Context, userID uint, voice *VoiceMessage) (*models.Reminder, error) {
	// 1. 处理语音消息
	result, err := vih.voiceService.ProcessVoiceMessage(ctx, voice)
	if err != nil {
		return nil, err
	}

	vih.log.Infof("语音处理结果: text='%s', confidence=%.2f", result.Text, result.Confidence)

	// 2. 如果有解析结果，创建提醒
	if result.ParseResult != nil && result.ParseResult.Intent == ai.IntentReminder && result.ParseResult.Reminder != nil {
		reminder := &models.Reminder{
			UserID:          userID,
			Title:           result.ParseResult.Reminder.Title,
			Description:     result.ParseResult.Reminder.Description,
			Type:            result.ParseResult.Reminder.Type,
			SchedulePattern: string(result.ParseResult.Reminder.SchedulePattern),
			TargetTime:      fmt.Sprintf("%02d:%02d:00", result.ParseResult.Reminder.Time.Hour, result.ParseResult.Reminder.Time.Minute),
			IsActive:        true,
		}

		if err := vih.reminderSvc.CreateReminder(ctx, reminder); err != nil {
			return nil, err
		}

		vih.log.Infof("从语音创建提醒成功: title='%s'", reminder.Title)
		return reminder, nil
	}

	return nil, fmt.Errorf("无法从语音中解析提醒内容")
}

// SupportedLanguages 获取支持的语言
func (vps *DefaultVoiceProcessingService) SupportedLanguages() []string {
	return []string{
		"zh-CN", // 中文
		"en-US", // 英语
		"ja-JP", // 日语
		"ko-KR", // 韩语
	}
}

// IsLanguageSupported 检查语言是否支持
func (vps *DefaultVoiceProcessingService) IsLanguageSupported(languageCode string) bool {
	for _, lang := range vps.SupportedLanguages() {
		if lang == languageCode {
			return true
		}
	}
	return false
}
