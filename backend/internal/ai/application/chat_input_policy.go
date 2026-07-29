package application

import (
	"encoding/base64"
	"strings"

	aidomain "k8s-platform-backend/internal/ai/domain"
)

const (
	chatInputMaxImages       = 6
	chatImageDataURLMaxBytes = 8 * 1024 * 1024
)

// ChatImageInput is the transport-neutral representation of an inline chat
// image. Runtime adapters can use it without coupling the application layer
// to a particular HTTP or gateway implementation.
type ChatImageInput struct {
	Name        string `json:"name"`
	ContentType string `json:"content_type"`
	DataURL     string `json:"data_url"`
	Size        int64  `json:"size"`
}

// ChatImage is the normalized image context that can be persisted with a
// message or converted by a gateway adapter into a provider-specific input.
type ChatImage struct {
	Name        string
	ContentType string
	DataURL     string
	Size        int64
}

// ChatFileContext contains the usable text portion of a normalized upload.
// Binary image data deliberately never enters this context.
type ChatFileContext struct {
	Name        string
	ContentType string
	Content     string
	Size        int64
}

// ChatInputPreparation keeps all deterministic chat-input decisions together
// before the retained runtime persists uploads or calls an AI gateway.
type ChatInputPreparation struct {
	Uploads []AIChatUploadInput
	Images  []ChatImage
	Files   []ChatFileContext
}

// PrepareChatInputs validates inline image data URLs, merges them with
// multipart uploads and creates provider-neutral image and text contexts.
func PrepareChatInputs(images []ChatImageInput, uploads []AIChatUploadInput) (ChatInputPreparation, error) {
	inlineUploads, err := inlineChatImagesToUploads(images)
	if err != nil {
		return ChatInputPreparation{}, err
	}

	preparedUploads := append([]AIChatUploadInput(nil), uploads...)
	preparedUploads = append(preparedUploads, inlineUploads...)

	normalizedImages := NormalizeChatImages(images)
	normalizedImages = append(normalizedImages, NormalizeChatUploadImages(uploads)...)
	return ChatInputPreparation{
		Uploads: preparedUploads,
		Images:  normalizedImages,
		Files:   BuildChatFileContexts(preparedUploads),
	}, nil
}

func inlineChatImagesToUploads(images []ChatImageInput) ([]AIChatUploadInput, error) {
	if len(images) == 0 {
		return nil, nil
	}

	out := make([]AIChatUploadInput, 0, len(images))
	for _, image := range images {
		dataURL := strings.TrimSpace(image.DataURL)
		if dataURL == "" {
			continue
		}
		contentType, raw, err := decodeChatImageDataURL(dataURL)
		if err != nil {
			return nil, ErrWithMessage(ErrInvalidParams, "图片附件格式无效")
		}
		if !strings.HasPrefix(strings.ToLower(contentType), "image/") {
			return nil, ErrWithMessage(ErrInvalidParams, "仅支持图片 data url")
		}
		name := strings.TrimSpace(image.Name)
		if name == "" {
			name = "image"
		}
		out = append(out, AIChatUploadInput{
			Name:        name,
			ContentType: strings.TrimSpace(image.ContentType),
			Size:        maxInt64(image.Size, int64(len(raw))),
			FileKind:    aiUploadKindImage,
			Bytes:       raw,
			DataURL:     dataURL,
		})
	}
	return out, nil
}

func decodeChatImageDataURL(dataURL string) (string, []byte, error) {
	trimmed := strings.TrimSpace(dataURL)
	if !strings.HasPrefix(strings.ToLower(trimmed), "data:") {
		return "", nil, ErrWithMessage(ErrInvalidParams, "invalid data url")
	}
	parts := strings.SplitN(trimmed, ",", 2)
	if len(parts) != 2 {
		return "", nil, ErrWithMessage(ErrInvalidParams, "invalid data url")
	}
	meta := strings.TrimPrefix(parts[0], "data:")
	metaParts := strings.Split(meta, ";")
	if len(metaParts) == 0 {
		return "", nil, ErrWithMessage(ErrInvalidParams, "invalid data url")
	}
	contentType := strings.TrimSpace(metaParts[0])
	if !strings.Contains(strings.ToLower(parts[0]), ";base64") {
		return "", nil, ErrWithMessage(ErrInvalidParams, "only base64 data url is supported")
	}
	raw, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return "", nil, err
	}
	return contentType, raw, nil
}

// NormalizeChatImages applies the persisted-message image contract to inline
// image inputs. Invalid optional image metadata is ignored, while malformed
// non-empty data URLs are rejected earlier by PrepareChatInputs.
func NormalizeChatImages(images []ChatImageInput) []ChatImage {
	if len(images) == 0 {
		return nil
	}

	capacity := len(images)
	if capacity > chatInputMaxImages {
		capacity = chatInputMaxImages
	}
	out := make([]ChatImage, 0, capacity)
	for _, item := range images {
		if len(out) >= chatInputMaxImages {
			break
		}
		dataURL := strings.TrimSpace(item.DataURL)
		contentType := strings.TrimSpace(item.ContentType)
		name := strings.TrimSpace(item.Name)
		if dataURL == "" || !strings.HasPrefix(strings.ToLower(dataURL), "data:image/") {
			continue
		}
		if contentType == "" {
			contentType = inferChatImageContentType(dataURL)
		}
		if !strings.HasPrefix(strings.ToLower(contentType), "image/") {
			continue
		}
		if len(dataURL) > chatImageDataURLMaxBytes {
			continue
		}
		if name == "" {
			name = "image"
		}
		out = append(out, ChatImage{
			Name:        name,
			ContentType: contentType,
			DataURL:     dataURL,
			Size:        maxInt64(item.Size, 0),
		})
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// NormalizeChatUploadImages includes persisted image uploads that have an
// inline data URL. This preserves the existing message snapshot behaviour for
// both multipart and inline image paths.
func NormalizeChatUploadImages(uploads []AIChatUploadInput) []ChatImage {
	if len(uploads) == 0 {
		return nil
	}

	out := make([]ChatImage, 0, len(uploads))
	for _, upload := range uploads {
		if upload.FileKind != aiUploadKindImage || strings.TrimSpace(upload.DataURL) == "" {
			continue
		}
		out = append(out, ChatImage{
			Name:        strings.TrimSpace(upload.Name),
			ContentType: strings.TrimSpace(upload.ContentType),
			DataURL:     strings.TrimSpace(upload.DataURL),
			Size:        maxInt64(upload.Size, int64(len(upload.Bytes))),
		})
	}
	return out
}

func BuildChatFileContexts(uploads []AIChatUploadInput) []ChatFileContext {
	if len(uploads) == 0 {
		return nil
	}
	out := make([]ChatFileContext, 0, len(uploads))
	for _, upload := range uploads {
		if upload.FileKind != aiUploadKindText {
			continue
		}
		out = append(out, ChatFileContext{
			Name:        strings.TrimSpace(upload.Name),
			ContentType: strings.TrimSpace(upload.ContentType),
			Content:     strings.TrimSpace(upload.TextContent),
			Size:        maxInt64(upload.Size, int64(len(upload.Bytes))),
		})
	}
	return out
}

func ChatImageSnapshots(images []ChatImage) []aidomain.JSONMap {
	if len(images) == 0 {
		return nil
	}
	out := make([]aidomain.JSONMap, 0, len(images))
	for _, image := range images {
		out = append(out, aidomain.JSONMap{
			"name":         image.Name,
			"content_type": image.ContentType,
			"data_url":     image.DataURL,
			"size":         image.Size,
		})
	}
	return out
}

// ValidateChatModelInputs enforces the capability contract after the runtime
// resolves an invocation target but before any gateway call is made.
func ValidateChatModelInputs(aiModel aidomain.AIModel, images []ChatImage, files []ChatFileContext) error {
	if len(images) > 0 && !aiModel.SupportsVision {
		return ErrWithMessage(ErrInvalidParams, "当前模型不支持图片输入，请切换到支持视觉的模型")
	}
	if len(files) > 0 && !aiModel.SupportsFileInput {
		return ErrWithMessage(ErrInvalidParams, "当前模型不支持文件输入，请切换到支持文件输入的模型")
	}
	return nil
}

func inferChatImageContentType(dataURL string) string {
	trimmed := strings.TrimSpace(dataURL)
	if !strings.HasPrefix(strings.ToLower(trimmed), "data:") {
		return ""
	}
	trimmed = strings.TrimPrefix(trimmed, "data:")
	parts := strings.SplitN(trimmed, ";", 2)
	return strings.TrimSpace(parts[0])
}
