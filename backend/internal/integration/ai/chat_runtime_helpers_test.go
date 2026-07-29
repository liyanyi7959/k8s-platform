package ai

import (
	"testing"

	aiapp "k8s-platform-backend/internal/ai/application"
	aidomain "k8s-platform-backend/internal/ai/domain"
)

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

func TestNormalizeAIModelAnswer(t *testing.T) {
	t.Run("empty input returns fallback", func(t *testing.T) {
		content, structured := aiapp.NormalizeModelAnswer("")
		if len(content) == 0 {
			t.Errorf("normalizeAIModelAnswer('') should return non-empty fallback content")
		}
		if structured == nil {
			t.Errorf("normalizeAIModelAnswer('') should return non-nil structured with guard info")
		}
	})

	t.Run("plain text", func(t *testing.T) {
		content, structured := aiapp.NormalizeModelAnswer("hello world")
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
		got := aiapp.MergeStructuredPayload(nil, nil)
		if got != nil {
			t.Errorf("mergeAIStructuredPayload(nil, nil) = %v, want nil", got)
		}
	})

	t.Run("first nil", func(t *testing.T) {
		base := aidomain.JSONMap{"key": "value"}
		got := aiapp.MergeStructuredPayload(nil, base)
		if got == nil || got["key"] != "value" {
			t.Errorf("mergeAIStructuredPayload(nil, base) = %v, want base", got)
		}
	})

	t.Run("second nil", func(t *testing.T) {
		extra := aidomain.JSONMap{"foo": "bar"}
		got := aiapp.MergeStructuredPayload(extra, nil)
		if got == nil || got["foo"] != "bar" {
			t.Errorf("mergeAIStructuredPayload(extra, nil) = %v, want extra", got)
		}
	})

	t.Run("merge with override", func(t *testing.T) {
		extra := aidomain.JSONMap{"foo": "bar", "extra_only": "yes"}
		base := aidomain.JSONMap{"foo": "old", "baz": "qux"}
		got := aiapp.MergeStructuredPayload(extra, base)
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
