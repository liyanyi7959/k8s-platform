package application

import (
	"errors"
	"testing"

	aidomain "k8s-platform-backend/internal/ai/domain"
)

func TestPrepareChatInputsBuildsProviderNeutralContexts(t *testing.T) {
	prepared, err := PrepareChatInputs(
		[]ChatImageInput{{Name: "screen.png", ContentType: "image/png", DataURL: "data:image/png;base64,aGVsbG8=", Size: 1}},
		[]AIChatUploadInput{
			{Name: "events.log", ContentType: "text/plain", FileKind: UploadKindText, Bytes: []byte("warning"), TextContent: " warning ", Size: 2},
			{Name: "stored.png", ContentType: "image/png", FileKind: UploadKindImage, Bytes: []byte("image"), DataURL: "data:image/png;base64,aW1hZ2U="},
		},
	)
	if err != nil {
		t.Fatalf("PrepareChatInputs() error = %v", err)
	}
	if len(prepared.Uploads) != 3 {
		t.Fatalf("uploads = %d, want 3", len(prepared.Uploads))
	}
	inline := prepared.Uploads[2]
	if inline.Name != "screen.png" || inline.FileKind != UploadKindImage || string(inline.Bytes) != "hello" || inline.Size != 5 {
		t.Fatalf("inline upload = %#v, want decoded image upload", inline)
	}
	if len(prepared.Images) != 2 {
		t.Fatalf("images = %#v, want inline and existing upload image", prepared.Images)
	}
	if prepared.Images[0].Name != "screen.png" || prepared.Images[0].ContentType != "image/png" {
		t.Fatalf("first image = %#v", prepared.Images[0])
	}
	if prepared.Images[1].Name != "stored.png" || prepared.Images[1].Size != 5 {
		t.Fatalf("second image = %#v", prepared.Images[1])
	}
	if len(prepared.Files) != 1 || prepared.Files[0].Name != "events.log" || prepared.Files[0].Content != "warning" || prepared.Files[0].Size != 7 {
		t.Fatalf("file contexts = %#v", prepared.Files)
	}

	snapshots := ChatImageSnapshots(prepared.Images)
	if len(snapshots) != 2 || snapshots[0]["name"] != "screen.png" || snapshots[1]["size"] != int64(5) {
		t.Fatalf("snapshots = %#v", snapshots)
	}
}

func TestPrepareChatInputsRejectsUnsupportedInlineImageData(t *testing.T) {
	_, err := PrepareChatInputs([]ChatImageInput{{DataURL: "data:text/plain;base64,aGVsbG8="}}, nil)
	if !errors.Is(err, ErrInvalidParams) {
		t.Fatalf("error = %v, want ErrInvalidParams", err)
	}
	if got := err.Error(); got != "仅支持图片 data url" {
		t.Fatalf("message = %q, want image data URL validation message", got)
	}
}

func TestNormalizeChatImagesLimitsAndFiltersOptionalMetadata(t *testing.T) {
	images := []ChatImageInput{
		{DataURL: "not-a-data-url"},
		{DataURL: "data:text/plain;base64,aGVsbG8="},
		{DataURL: "data:image/jpeg;base64,aGVsbG8="},
	}
	for i := 0; i < 8; i++ {
		images = append(images, ChatImageInput{Name: "image", ContentType: "image/png", DataURL: "data:image/png;base64,aGVsbG8="})
	}

	normalized := NormalizeChatImages(images)
	if len(normalized) != chatInputMaxImages {
		t.Fatalf("normalized images = %d, want limit %d", len(normalized), chatInputMaxImages)
	}
	if normalized[0].Name != "image" || normalized[0].ContentType != "image/jpeg" {
		t.Fatalf("inferred image = %#v", normalized[0])
	}
}

func TestValidateChatModelInputs(t *testing.T) {
	image := []ChatImage{{Name: "screen.png"}}
	file := []ChatFileContext{{Name: "events.log"}}

	err := ValidateChatModelInputs(aidomain.AIModel{}, image, nil)
	if !errors.Is(err, ErrInvalidParams) || err.Error() != "当前模型不支持图片输入，请切换到支持视觉的模型" {
		t.Fatalf("vision error = %v", err)
	}
	err = ValidateChatModelInputs(aidomain.AIModel{SupportsVision: true}, nil, file)
	if !errors.Is(err, ErrInvalidParams) || err.Error() != "当前模型不支持文件输入，请切换到支持文件输入的模型" {
		t.Fatalf("file input error = %v", err)
	}
	if err := ValidateChatModelInputs(aidomain.AIModel{SupportsVision: true, SupportsFileInput: true}, image, file); err != nil {
		t.Fatalf("supported model error = %v", err)
	}
}
