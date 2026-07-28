package service

import (
	"context"
	"errors"
	"testing"

	"k8s-platform-backend/internal/legacy/model"
)

func TestMaxInt64(t *testing.T) {
	tests := []struct {
		name  string
		value int64
		floor int64
		want  int64
	}{
		{"value above floor", 10, 5, 10},
		{"value below floor", 3, 5, 5},
		{"value equal floor", 5, 5, 5},
		{"negative values", -3, -5, -3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := maxInt64(tt.value, tt.floor); got != tt.want {
				t.Errorf("maxInt64(%d, %d) = %d, want %d", tt.value, tt.floor, got, tt.want)
			}
		})
	}
}

func TestAIConversationRunStatusFromError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{"nil error", nil, "failed"},
		{"context canceled", context.Canceled, "cancelled"},
		{"deadline exceeded", context.DeadlineExceeded, "failed"},
		{"other error", errors.New("test error"), "failed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := aiConversationRunStatusFromError(tt.err); got != tt.want {
				t.Errorf("aiConversationRunStatusFromError(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

func TestAIMessageStatusFromError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{"nil error", nil, "failed"},
		{"context canceled", context.Canceled, "cancelled"},
		{"deadline exceeded", context.DeadlineExceeded, "failed"},
		{"other error", errors.New("test error"), "failed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := aiMessageStatusFromError(tt.err); got != tt.want {
				t.Errorf("aiMessageStatusFromError(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

func TestBuildAIAssistantFailureReply(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{"context canceled", context.Canceled, "本轮回答已取消，平台已停止继续生成结果。你可以调整问题或范围后重新发送。"},
		{"deadline exceeded", context.DeadlineExceeded, "本轮回答处理超时，AI 未能在限定时间内完成。建议缩小排查范围后重试。"},
		{"other error", errors.New("test error"), "本轮回答未成功完成。原因：test error"},
		{"empty error", errors.New(""), "本轮回答未成功完成。原因：本轮回答失败，请稍后重试。"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := buildAIAssistantFailureReply(tt.err); got != tt.want {
				t.Errorf("buildAIAssistantFailureReply(%v) = %q, want %q", tt.err, got, tt.want)
			}
		})
	}
}

func TestNormalizeAIChatImages(t *testing.T) {
	t.Run("nil images", func(t *testing.T) {
		got := normalizeAIChatImages(nil)
		if len(got) != 0 {
			t.Errorf("normalizeAIChatImages(nil) returned %d items, want 0", len(got))
		}
	})

	t.Run("empty images", func(t *testing.T) {
		got := normalizeAIChatImages([]AIChatImageInput{})
		if len(got) != 0 {
			t.Errorf("normalizeAIChatImages([]) returned %d items, want 0", len(got))
		}
	})

	t.Run("valid image", func(t *testing.T) {
		got := normalizeAIChatImages([]AIChatImageInput{
			{DataURL: "data:image/png;base64,abc123", ContentType: "image/png", Size: 100},
		})
		if len(got) != 1 {
			t.Errorf("normalizeAIChatImages() returned %d items, want 1", len(got))
		}
	})
}

func TestBuildConversationSummary(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{"empty content", ""},
		{"short content", "Hello"},
		{"long content", string(make([]byte, 300))},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildConversationSummary(tt.content)
			if len(got) > 200 {
				t.Errorf("buildConversationSummary() returned string of length %d, max 200", len(got))
			}
		})
	}
}

func TestPtrUint64(t *testing.T) {
	got := ptrUint64(42)
	if got == nil || *got != 42 {
		t.Errorf("ptrUint64(42) = %v, want 42", got)
	}

	got0 := ptrUint64(0)
	if got0 == nil || *got0 != 0 {
		t.Errorf("ptrUint64(0) = %v, want 0", got0)
	}
}

func TestFirstUserFacingError(t *testing.T) {
	t.Run("nil error", func(t *testing.T) {
		if got := firstUserFacingError(nil); got != "" {
			t.Errorf("firstUserFacingError(nil) = %q, want empty", got)
		}
	})

	t.Run("plain error", func(t *testing.T) {
		if got := firstUserFacingError(errors.New("test message")); got != "test message" {
			t.Errorf("firstUserFacingError() = %q, want %q", got, "test message")
		}
	})
}

func TestNormalizeAIModelAnswer(t *testing.T) {
	t.Run("empty input returns fallback", func(t *testing.T) {
		content, structured := normalizeAIModelAnswer("")
		if len(content) == 0 {
			t.Errorf("normalizeAIModelAnswer('') should return non-empty fallback content")
		}
		if structured == nil {
			t.Errorf("normalizeAIModelAnswer('') should return non-nil structured with guard info")
		}
	})

	t.Run("plain text", func(t *testing.T) {
		content, structured := normalizeAIModelAnswer("hello world")
		if content != "hello world" {
			t.Errorf("normalizeAIModelAnswer() content = %q, want %q", content, "hello world")
		}
		if structured != nil {
			t.Errorf("normalizeAIModelAnswer() structured = %v, want nil", structured)
		}
	})
}

func TestMergeAIStructuredPayload(t *testing.T) {
	t.Run("both nil", func(t *testing.T) {
		got := mergeAIStructuredPayload(nil, nil)
		if got != nil {
			t.Errorf("mergeAIStructuredPayload(nil, nil) = %v, want nil", got)
		}
	})

	t.Run("first nil", func(t *testing.T) {
		base := model.JSONMap{"key": "value"}
		got := mergeAIStructuredPayload(nil, base)
		if got == nil || got["key"] != "value" {
			t.Errorf("mergeAIStructuredPayload(nil, base) = %v, want base", got)
		}
	})

	t.Run("second nil", func(t *testing.T) {
		extra := model.JSONMap{"foo": "bar"}
		got := mergeAIStructuredPayload(extra, nil)
		if got == nil || got["foo"] != "bar" {
			t.Errorf("mergeAIStructuredPayload(extra, nil) = %v, want extra", got)
		}
	})

	t.Run("merge with override", func(t *testing.T) {
		extra := model.JSONMap{"foo": "bar", "extra_only": "yes"}
		base := model.JSONMap{"foo": "old", "baz": "qux"}
		got := mergeAIStructuredPayload(extra, base)
		// base (second arg) overrides extra (first arg) for same keys
		if got["foo"] != "old" {
			t.Errorf("mergeAIStructuredPayload() foo = %v, want old", got["foo"])
		}
		if got["baz"] != "qux" {
			t.Errorf("mergeAIStructuredPayload() baz = %v, want qux", got["baz"])
		}
		if got["extra_only"] != "yes" {
			t.Errorf("mergeAIStructuredPayload() extra_only = %v, want yes", got["extra_only"])
		}
	})
}
