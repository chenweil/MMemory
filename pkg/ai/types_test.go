package ai

import (
	"testing"
	"time"

	"mmemory/internal/models"
)

func TestParseResultValidate_NewIntents(t *testing.T) {
	tests := []struct {
		name       string
		result     *ParseResult
		wantValid  bool
		wantErrors []string
	}{
		{
			name: "delete intent with keywords",
			result: &ParseResult{
				Intent:     IntentDelete,
				Confidence: 0.9,
				Delete: &DeleteInfo{
					Keywords: []string{"健身", "晚上"},
				},
			},
			wantValid: true,
		},
		{
			name: "delete intent missing details",
			result: &ParseResult{
				Intent:     IntentDelete,
				Confidence: 0.9,
				Delete: &DeleteInfo{
					Keywords: []string{},
					Criteria: "",
				},
			},
			wantValid:  false,
			wantErrors: []string{"delete keywords or criteria required"},
		},
		{
			name: "edit intent requires update fields",
			result: &ParseResult{
				Intent:     IntentEdit,
				Confidence: 0.9,
				Edit: &EditInfo{
					Keywords: []string{"健身"},
				},
			},
			wantValid:  false,
			wantErrors: []string{"edit requires at least one field to update"},
		},
		{
			name: "pause intent valid",
			result: &ParseResult{
				Intent:     IntentPause,
				Confidence: 0.8,
				Pause: &PauseInfo{
					Keywords: []string{"健身"},
					Duration: "P1W",
				},
			},
			wantValid: true,
		},
		{
			name: "resume intent missing keywords",
			result: &ParseResult{
				Intent:     IntentResume,
				Confidence: 0.8,
				Resume: &ResumeInfo{
					Keywords: []string{""},
				},
			},
			wantValid:  false,
			wantErrors: []string{"resume keywords required"},
		},
		{
			name: "reminder intent still valid",
			result: &ParseResult{
				Intent:     IntentReminder,
				Confidence: 0.95,
				Reminder: &ReminderInfo{
					Title: "喝水",
					Type:  models.ReminderTypeHabit,
					Time: TimeInfo{
						Hour:     8,
						Minute:   0,
						Timezone: "Asia/Shanghai",
					},
					SchedulePattern: models.SchedulePatternDaily,
				},
				Timestamp: time.Now(),
			},
			wantValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.result.Validate()
			if result.IsValid != tt.wantValid {
				t.Fatalf("Validate().IsValid = %v, want %v (errors: %v)", result.IsValid, tt.wantValid, result.Errors)
			}
			if !tt.wantValid {
				for _, want := range tt.wantErrors {
					found := false
					for _, got := range result.Errors {
						if got == want {
							found = true
							break
						}
					}
					if !found {
						t.Fatalf("expected error %q not found in %v", want, result.Errors)
					}
				}
			}
		})
	}
}
