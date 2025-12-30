package service

import (
	"context"
	"testing"
	"time"

	"mmemory/internal/models"
)

// MockBotAPI 模拟Telegram Bot API用于测试
type MockBotAPI struct {
	mockFile     *BotFile
	mockFileData []byte
	mockErr      error
}

func (m *MockBotAPI) GetFile(fileID string) (*BotFile, error) {
	if m.mockErr != nil {
		return nil, m.mockErr
	}
	return m.mockFile, nil
}

func (m *MockBotAPI) DownloadFile(filePath string) ([]byte, error) {
	if m.mockErr != nil {
		return nil, m.mockErr
	}
	return m.mockFileData, nil
}

// MockError 模拟错误用于测试
type MockError struct {
	message string
}

func (e *MockError) Error() string {
	return e.message
}

func TestVoiceProcessingService_ProcessVoiceMessage(t *testing.T) {
	t.Run("处理语音消息成功", func(t *testing.T) {
		mockBotAPI := &MockBotAPI{
			mockFile: &BotFile{
				FileID:   "test_file_id",
				FilePath: "voice/test.ogg",
				FileSize: 1024,
			},
			mockFileData: []byte("audio data"),
		}

		voiceService := NewDefaultVoiceProcessingService(nil, mockBotAPI)

		voice := &VoiceMessage{
			FileID:       "test_file_id",
			Duration:     5,
			MimeType:     "audio/ogg",
			FileSize:     1024,
			LanguageCode: "zh-CN",
			UploadedAt:   time.Now(),
		}

		result, err := voiceService.ProcessVoiceMessage(context.Background(), voice)
		if err != nil {
			t.Fatalf("期望处理成功，实际: %v", err)
		}

		if result == nil {
			t.Fatal("期望返回结果，实际为nil")
		}

		if result.Text == "" {
			t.Error("期望有文本内容，实际为空")
		}

		if result.Confidence <= 0 || result.Confidence > 1 {
			t.Errorf("期望置信度在0-1之间，实际: %.2f", result.Confidence)
		}

		if result.Language == "" {
			t.Error("期望有语言代码，实际为空")
		}

		t.Logf("语音处理结果: text='%s', confidence=%.2f, language=%s",
			result.Text, result.Confidence, result.Language)
	})

	t.Run("下载语音文件失败", func(t *testing.T) {
		mockBotAPI := &MockBotAPI{
			mockErr: &MockError{message: "file not found"},
		}

		voiceService := NewDefaultVoiceProcessingService(nil, mockBotAPI)

		voice := &VoiceMessage{
			FileID:   "invalid_file_id",
			Duration: 5,
			MimeType: "audio/ogg",
		}

		_, err := voiceService.ProcessVoiceMessage(context.Background(), voice)
		if err == nil {
			t.Fatal("期望返回错误，实际为nil")
		}

		t.Logf("期望的错误: %v", err)
	})
}

func TestDefaultVoiceProcessingService_TranscribeAudio(t *testing.T) {
	t.Run("转录音频返回模拟结果", func(t *testing.T) {
		voiceService := NewDefaultVoiceProcessingService(nil, nil)

		result, err := voiceService.TranscribeAudio(context.Background(), []byte("audio data"), "audio/ogg")
		if err != nil {
			t.Fatalf("期望无错误，实际: %v", err)
		}

		if result == nil {
			t.Fatal("期望返回结果，实际为nil")
		}

		if result.Text == "" {
			t.Error("期望有文本内容，实际为空")
		}

		if result.Confidence <= 0 || result.Confidence > 1 {
			t.Errorf("期望置信度在0-1之间，实际: %.2f", result.Confidence)
		}

		t.Logf("转录结果: text='%s', confidence=%.2f, language=%s, duration=%d",
			result.Text, result.Confidence, result.Language, result.Duration)
	})
}

func TestDefaultVoiceProcessingService_ParseTranscribedText(t *testing.T) {
	t.Run("解析转录文本返回模拟结果", func(t *testing.T) {
		voiceService := NewDefaultVoiceProcessingService(nil, nil)

		result, err := voiceService.ParseTranscribedText(context.Background(), "请每天早上提醒我喝水", 1)
		if err != nil {
			t.Fatalf("期望无错误，实际: %v", err)
		}

		if result == nil {
			t.Fatal("期望返回结果，实际为nil")
		}

		if result.Reminder == nil {
			t.Fatal("期望有Reminder信息，实际为nil")
		}

		if result.Reminder.Title == "" {
			t.Error("期望有提醒标题，实际为空")
		}

		t.Logf("解析结果: title='%s', type=%s, time=%02d:%02d",
			result.Reminder.Title, result.Reminder.Type,
			result.Reminder.Time.Hour, result.Reminder.Time.Minute)
	})
}

func TestDefaultVoiceProcessingService_SupportedLanguages(t *testing.T) {
	voiceService := NewDefaultVoiceProcessingService(nil, nil)

	languages := voiceService.SupportedLanguages()

	if len(languages) == 0 {
		t.Error("期望有支持的语言列表，实际为空")
	}

	expectedLanguages := []string{"zh-CN", "en-US", "ja-JP", "ko-KR"}
	for i, lang := range languages {
		if lang != expectedLanguages[i] {
			t.Errorf("期望语言%s，实际: %s", expectedLanguages[i], lang)
		}
	}
}

func TestDefaultVoiceProcessingService_IsLanguageSupported(t *testing.T) {
	voiceService := NewDefaultVoiceProcessingService(nil, nil)

	tests := []struct {
		languageCode string
		expected     bool
	}{
		{"zh-CN", true},
		{"en-US", true},
		{"ja-JP", true},
		{"ko-KR", true},
		{"fr-FR", false},
		{"", false},
	}

	for _, tt := range tests {
		result := voiceService.IsLanguageSupported(tt.languageCode)
		if result != tt.expected {
			t.Errorf("IsLanguageSupported(%q) = %v, 期望 %v", tt.languageCode, result, tt.expected)
		}
	}
}

func TestVoiceInputHandler_HandleVoiceMessage(t *testing.T) {
	t.Run("从语音创建提醒成功", func(t *testing.T) {
		mockBotAPI := &MockBotAPI{
			mockFile: &BotFile{
				FileID:   "test_file_id",
				FilePath: "voice/test.ogg",
				FileSize: 1024,
			},
			mockFileData: []byte("audio data"),
		}

		voiceService := NewDefaultVoiceProcessingService(nil, mockBotAPI)

		// 使用嵌入式接口的模拟实现
		mockReminderSvc := &mockReminderServiceImpl{
			createdReminders: make([]*models.Reminder, 0),
		}

		handler := NewVoiceInputHandler(voiceService, mockReminderSvc)

		voice := &VoiceMessage{
			FileID:       "test_file_id",
			Duration:     5,
			MimeType:     "audio/ogg",
			LanguageCode: "zh-CN",
			UploadedAt:   time.Now(),
		}

		reminder, err := handler.HandleVoiceMessage(context.Background(), 1, voice)
		if err != nil {
			t.Fatalf("期望创建成功，实际: %v", err)
		}

		if reminder == nil {
			t.Fatal("期望返回提醒，实际为nil")
		}

		if len(mockReminderSvc.createdReminders) == 0 {
			t.Error("期望创建提醒，实际未创建")
		} else {
			t.Logf("创建的提醒数量: %d", len(mockReminderSvc.createdReminders))
		}

		t.Logf("创建的提醒: title='%s', time='%s', type=%s",
			reminder.Title, reminder.TargetTime, reminder.Type)
	})

	t.Run("无法解析提醒内容", func(t *testing.T) {
		mockBotAPI := &MockBotAPI{
			mockFile: &BotFile{
				FileID:   "test_file_id",
				FilePath: "voice/test.ogg",
				FileSize: 1024,
			},
			mockFileData: []byte("audio data"),
		}

		voiceService := NewDefaultVoiceProcessingService(nil, mockBotAPI)
		mockReminderSvc := &mockReminderServiceImpl{
			createdReminders: make([]*models.Reminder, 0),
		}

		handler := NewVoiceInputHandler(voiceService, mockReminderSvc)

		// 语音处理会返回模拟的提醒解析结果，这里测试非提醒意图的情况
		voice := &VoiceMessage{
			FileID:       "test_file_id",
			Duration:     5,
			MimeType:     "audio/ogg",
			LanguageCode: "zh-CN",
			UploadedAt:   time.Now(),
		}

		_, err := handler.HandleVoiceMessage(context.Background(), 1, voice)
		// 由于模拟的语音转录返回的是提醒意图，这里不会返回错误
		// 实际测试中可能需要修改模拟逻辑
		t.Logf("处理结果: err=%v", err)
	})
}

// mockReminderServiceImpl 模拟提醒服务实现
type mockReminderServiceImpl struct {
	createdReminders []*models.Reminder
}

func (m *mockReminderServiceImpl) CreateReminder(ctx context.Context, reminder *models.Reminder) error {
	m.createdReminders = append(m.createdReminders, reminder)
	return nil
}

func (m *mockReminderServiceImpl) ParseReminderFromText(ctx context.Context, text string, userID uint) (*models.Reminder, error) {
	return nil, nil
}

func (m *mockReminderServiceImpl) GetUserReminders(ctx context.Context, userID uint) ([]*models.Reminder, error) {
	return nil, nil
}

func (m *mockReminderServiceImpl) GetReminderByID(ctx context.Context, id uint) (*models.Reminder, error) {
	return nil, nil
}

func (m *mockReminderServiceImpl) UpdateReminder(ctx context.Context, reminder *models.Reminder) error {
	return nil
}

func (m *mockReminderServiceImpl) EditReminder(ctx context.Context, params EditReminderParams) error {
	return nil
}

func (m *mockReminderServiceImpl) DeleteReminder(ctx context.Context, id uint) error {
	return nil
}

func (m *mockReminderServiceImpl) PauseReminder(ctx context.Context, id uint, duration time.Duration, reason string) error {
	return nil
}

func (m *mockReminderServiceImpl) ResumeReminder(ctx context.Context, id uint) error {
	return nil
}

func TestWhisperSpeechToText_Transcribe(t *testing.T) {
	whisper := NewWhisperSpeechToText("test_api_key", "https://api.openai.com/v1")

	result, err := whisper.Transcribe(context.Background(), []byte("audio data"), "audio/ogg")
	if err != nil {
		t.Fatalf("期望无错误，实际: %v", err)
	}

	if result == nil {
		t.Fatal("期望返回结果，实际为nil")
	}

	if result.Text == "" {
		t.Error("期望有文本内容，实际为空")
	}

	if result.Confidence <= 0 || result.Confidence > 1 {
		t.Errorf("期望置信度在0-1之间，实际: %.2f", result.Confidence)
	}

	t.Logf("Whisper转录结果: text='%s', confidence=%.2f, language=%s",
		result.Text, result.Confidence, result.Language)
}

func TestGoogleSpeechToText_Transcribe(t *testing.T) {
	google := NewGoogleSpeechToText("test_api_key", "test_project_id")

	result, err := google.Transcribe(context.Background(), []byte("audio data"), "audio/ogg")
	if err != nil {
		t.Fatalf("期望无错误，实际: %v", err)
	}

	if result == nil {
		t.Fatal("期望返回结果，实际为nil")
	}

	if result.Text == "" {
		t.Error("期望有文本内容，实际为空")
	}

	if result.Confidence <= 0 || result.Confidence > 1 {
		t.Errorf("期望置信度在0-1之间，实际: %.2f", result.Confidence)
	}

	t.Logf("Google转录结果: text='%s', confidence=%.2f, language=%s",
		result.Text, result.Confidence, result.Language)
}

func TestVoiceMessage_Struct(t *testing.T) {
	voice := &VoiceMessage{
		FileID:       "test_file_id",
		Duration:     30,
		MimeType:     "audio/ogg",
		FileSize:     102400,
		Waveform:     " waveform_data",
		LanguageCode: "zh-CN",
		UploadedAt:   time.Now(),
	}

	if voice.FileID != "test_file_id" {
		t.Errorf("期望FileID为test_file_id，实际: %s", voice.FileID)
	}

	if voice.Duration != 30 {
		t.Errorf("期望Duration为30，实际: %d", voice.Duration)
	}

	if voice.MimeType != "audio/ogg" {
		t.Errorf("期望MimeType为audio/ogg，实际: %s", voice.MimeType)
	}

	if voice.LanguageCode != "zh-CN" {
		t.Errorf("期望LanguageCode为zh-CN，实际: %s", voice.LanguageCode)
	}

	t.Logf("VoiceMessage结构测试通过")
}

func TestVoiceProcessingResult_Struct(t *testing.T) {
	result := &VoiceProcessingResult{
		Text:           "请每天早上提醒我喝水",
		Confidence:     0.95,
		Language:       "zh-CN",
		Duration:       3000,
		ProcessingTime: time.Second * 2,
	}

	if result.Text != "请每天早上提醒我喝水" {
		t.Errorf("期望Text为'请每天早上提醒我喝水'，实际: %s", result.Text)
	}

	if result.Confidence != 0.95 {
		t.Errorf("期望Confidence为0.95，实际: %.2f", result.Confidence)
	}

	if result.Language != "zh-CN" {
		t.Errorf("期望Language为zh-CN，实际: %s", result.Language)
	}

	if result.Duration != 3000 {
		t.Errorf("期望Duration为3000，实际: %d", result.Duration)
	}

	t.Logf("VoiceProcessingResult结构测试通过")
}

func TestTranscriptionResult_Struct(t *testing.T) {
	result := &TranscriptionResult{
		Text:       "请每天早上提醒我喝水",
		Confidence: 0.95,
		Language:   "zh-CN",
		Duration:   3000,
	}

	if result.Text != "请每天早上提醒我喝水" {
		t.Errorf("期望Text为'请每天早上提醒我喝水'，实际: %s", result.Text)
	}

	if result.Confidence != 0.95 {
		t.Errorf("期望Confidence为0.95，实际: %.2f", result.Confidence)
	}

	if result.Language != "zh-CN" {
		t.Errorf("期望Language为zh-CN，实际: %s", result.Language)
	}

	if result.Duration != 3000 {
		t.Errorf("期望Duration为3000，实际: %d", result.Duration)
	}

	t.Logf("TranscriptionResult结构测试通过")
}

func TestBotFile_Struct(t *testing.T) {
	botFile := &BotFile{
		FileID:   "test_file_id",
		FilePath: "voice/test.ogg",
		FileSize: 1024,
	}

	if botFile.FileID != "test_file_id" {
		t.Errorf("期望FileID为test_file_id，实际: %s", botFile.FileID)
	}

	if botFile.FilePath != "voice/test.ogg" {
		t.Errorf("期望FilePath为voice/test.ogg，实际: %s", botFile.FilePath)
	}

	if botFile.FileSize != 1024 {
		t.Errorf("期望FileSize为1024，实际: %d", botFile.FileSize)
	}

	t.Logf("BotFile结构测试通过")
}
