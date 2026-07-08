package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"gorm.io/gorm"

	"k8s-platform-backend/internal/model"
)

const (
	defaultAIUploadDir       = "data/ai-uploads"
	aiChatMaxAttachmentCount = 6
	aiChatMaxImageBytes      = 5 * 1024 * 1024
	aiChatMaxTextBytes       = 512 * 1024
	aiChatMaxFileTextRunes   = 12000
)

const (
	aiUploadKindImage = "image"
	aiUploadKindText  = "text"
)

type AIChatUploadInput struct {
	Name        string
	ContentType string
	Size        int64
	FileKind    string
	Bytes       []byte
	DataURL     string
	TextContent string
}

type AIMessageAttachmentItem struct {
	ID             uint64  `json:"id"`
	ConversationID *uint64 `json:"conversation_id,omitempty"`
	MessageID      *uint64 `json:"message_id,omitempty"`
	OriginalName   string  `json:"original_name"`
	ContentType    string  `json:"content_type"`
	FileSize       int64   `json:"file_size"`
	Purpose        string  `json:"purpose"`
	Status         string  `json:"status"`
	FileKind       string  `json:"file_kind"`
	DownloadURL    string  `json:"download_url"`
	CreatedAt      string  `json:"created_at"`
}

type AIFileService struct {
	db      *gorm.DB
	baseDir string
}

func NewAIFileService(db *gorm.DB, baseDir string) *AIFileService {
	baseDir = strings.TrimSpace(baseDir)
	if baseDir == "" {
		baseDir = defaultAIUploadDir
	}
	return &AIFileService{db: db, baseDir: baseDir}
}

func (s *AIFileService) NormalizeChatUploads(files []*multipart.FileHeader) ([]AIChatUploadInput, error) {
	if len(files) == 0 {
		return nil, nil
	}
	if len(files) > aiChatMaxAttachmentCount {
		return nil, ErrWithMessage(ErrInvalidParams, fmt.Sprintf("单次最多上传 %d 个附件", aiChatMaxAttachmentCount))
	}

	out := make([]AIChatUploadInput, 0, len(files))
	for _, file := range files {
		item, err := normalizeAIChatUpload(file)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, nil
}

func (s *AIFileService) SaveChatUploads(
	ctx context.Context,
	conversationID,
	messageID,
	userID uint64,
	username string,
	uploads []AIChatUploadInput,
) ([]AIMessageAttachmentItem, error) {
	if len(uploads) == 0 {
		return nil, nil
	}
	if s == nil || s.db == nil {
		return nil, errors.New("db is required")
	}

	baseDir, err := filepath.Abs(s.baseDir)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return nil, err
	}

	rows := make([]model.AIUploadedFile, 0, len(uploads))
	writtenPaths := make([]string, 0, len(uploads))
	for _, upload := range uploads {
		if len(upload.Bytes) == 0 {
			continue
		}
		storagePath, shaValue, err := writeAIUploadFile(baseDir, upload)
		if err != nil {
			cleanupAIUploadFiles(writtenPaths)
			return nil, err
		}
		writtenPaths = append(writtenPaths, storagePath)

		rows = append(rows, model.AIUploadedFile{
			ConversationID: &conversationID,
			MessageID:      &messageID,
			StoragePath:    storagePath,
			OriginalName:   upload.Name,
			ContentType:    upload.ContentType,
			FileSize:       int64(len(upload.Bytes)),
			Purpose:        "chat",
			Status:         "active",
			SHA256:         shaValue,
			CreatedBy:      userID,
			CreatedByName:  strings.TrimSpace(username),
		})
	}

	if len(rows) == 0 {
		return nil, nil
	}

	if err := s.db.WithContext(ctx).Create(&rows).Error; err != nil {
		cleanupAIUploadFiles(writtenPaths)
		return nil, err
	}

	items := make([]AIMessageAttachmentItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, buildAIMessageAttachmentItem(row))
	}
	return items, nil
}

func (s *AIFileService) ListMessageAttachments(ctx context.Context, conversationID uint64) (map[uint64][]AIMessageAttachmentItem, error) {
	return listConversationAttachments(ctx, s.db, conversationID)
}

func (s *AIFileService) OpenAttachment(ctx context.Context, id uint64) (AIMessageAttachmentItem, *os.File, error) {
	if s == nil || s.db == nil {
		return AIMessageAttachmentItem{}, nil, errors.New("db is required")
	}
	if id == 0 {
		return AIMessageAttachmentItem{}, nil, ErrWithMessage(ErrInvalidParams, "附件 ID 无效")
	}

	var row model.AIUploadedFile
	if err := s.db.WithContext(ctx).
		Where("deleted_at IS NULL AND id = ?", id).
		First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return AIMessageAttachmentItem{}, nil, ErrNotFound
		}
		return AIMessageAttachmentItem{}, nil, err
	}

	file, err := os.Open(filepath.Clean(row.StoragePath))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return AIMessageAttachmentItem{}, nil, ErrNotFound
		}
		return AIMessageAttachmentItem{}, nil, err
	}
	return buildAIMessageAttachmentItem(row), file, nil
}

func listConversationAttachments(ctx context.Context, db *gorm.DB, conversationID uint64) (map[uint64][]AIMessageAttachmentItem, error) {
	out := make(map[uint64][]AIMessageAttachmentItem)
	if db == nil || conversationID == 0 {
		return out, nil
	}

	var rows []model.AIUploadedFile
	if err := db.WithContext(ctx).
		Where("deleted_at IS NULL AND conversation_id = ?", conversationID).
		Order("created_at ASC, id ASC").
		Find(&rows).Error; err != nil {
		return nil, err
	}

	for _, row := range rows {
		if row.MessageID == nil || *row.MessageID == 0 {
			continue
		}
		messageID := *row.MessageID
		out[messageID] = append(out[messageID], buildAIMessageAttachmentItem(row))
	}
	return out, nil
}

func buildAIMessageAttachmentItem(row model.AIUploadedFile) AIMessageAttachmentItem {
	return AIMessageAttachmentItem{
		ID:             row.ID,
		ConversationID: row.ConversationID,
		MessageID:      row.MessageID,
		OriginalName:   row.OriginalName,
		ContentType:    row.ContentType,
		FileSize:       row.FileSize,
		Purpose:        row.Purpose,
		Status:         row.Status,
		FileKind:       detectAIStoredFileKind(row.OriginalName, row.ContentType),
		DownloadURL:    fmt.Sprintf("/api/v1/ai/files/%d/content", row.ID),
		CreatedAt:      row.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func normalizeAIChatUpload(file *multipart.FileHeader) (AIChatUploadInput, error) {
	if file == nil {
		return AIChatUploadInput{}, ErrWithMessage(ErrInvalidParams, "附件不能为空")
	}

	name := sanitizeAIUploadName(file.Filename)
	if name == "" {
		name = "attachment"
	}

	reader, err := file.Open()
	if err != nil {
		return AIChatUploadInput{}, err
	}
	defer func() { _ = reader.Close() }()

	limit := maxInt64(aiChatMaxImageBytes, aiChatMaxTextBytes) + 1
	data, err := io.ReadAll(io.LimitReader(reader, limit))
	if err != nil {
		return AIChatUploadInput{}, err
	}
	if int64(len(data)) > limit-1 {
		return AIChatUploadInput{}, ErrWithMessage(ErrInvalidParams, fmt.Sprintf("附件 %s 超过大小限制", name))
	}
	if len(data) == 0 {
		return AIChatUploadInput{}, ErrWithMessage(ErrInvalidParams, fmt.Sprintf("附件 %s 不能为空", name))
	}

	contentType := strings.TrimSpace(file.Header.Get("Content-Type"))
	if contentType == "" {
		contentType = strings.TrimSpace(mime.TypeByExtension(strings.ToLower(filepath.Ext(name))))
	}
	if contentType == "" {
		contentType = http.DetectContentType(data)
	}
	kind := detectAIChatUploadKind(name, contentType, data)
	if kind == "" {
		return AIChatUploadInput{}, ErrWithMessage(ErrInvalidParams, fmt.Sprintf("附件 %s 格式暂不支持", name))
	}

	switch kind {
	case aiUploadKindImage:
		if int64(len(data)) > aiChatMaxImageBytes {
			return AIChatUploadInput{}, ErrWithMessage(ErrInvalidParams, fmt.Sprintf("图片 %s 不能超过 %d MB", name, aiChatMaxImageBytes/1024/1024))
		}
		return AIChatUploadInput{
			Name:        name,
			ContentType: normalizeAIUploadContentType(contentType, http.DetectContentType(data)),
			Size:        int64(len(data)),
			FileKind:    aiUploadKindImage,
			Bytes:       data,
			DataURL:     buildAIUploadDataURL(contentType, data),
		}, nil
	case aiUploadKindText:
		if int64(len(data)) > aiChatMaxTextBytes {
			return AIChatUploadInput{}, ErrWithMessage(ErrInvalidParams, fmt.Sprintf("文本附件 %s 不能超过 %d KB", name, aiChatMaxTextBytes/1024))
		}
		if !utf8.Valid(data) {
			return AIChatUploadInput{}, ErrWithMessage(ErrInvalidParams, fmt.Sprintf("文本附件 %s 编码无效，请使用 UTF-8", name))
		}
		return AIChatUploadInput{
			Name:        name,
			ContentType: normalizeAIUploadContentType(contentType, http.DetectContentType(data)),
			Size:        int64(len(data)),
			FileKind:    aiUploadKindText,
			Bytes:       data,
			TextContent: string(data),
		}, nil
	default:
		return AIChatUploadInput{}, ErrWithMessage(ErrInvalidParams, fmt.Sprintf("附件 %s 格式暂不支持", name))
	}
}

func writeAIUploadFile(baseDir string, upload AIChatUploadInput) (string, string, error) {
	now := time.Now().UTC()
	dir := filepath.Join(baseDir, now.Format("200601"), now.Format("02"))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", "", err
	}

	randomSuffix, err := randomHex(10)
	if err != nil {
		return "", "", err
	}
	ext := strings.ToLower(filepath.Ext(upload.Name))
	filename := fmt.Sprintf("%d_%s%s", now.UnixNano(), randomSuffix, ext)
	storagePath := filepath.Join(dir, filename)

	if err := os.WriteFile(storagePath, upload.Bytes, 0o644); err != nil {
		return "", "", err
	}
	shaValue := sha256.Sum256(upload.Bytes)
	return storagePath, hex.EncodeToString(shaValue[:]), nil
}

func cleanupAIUploadFiles(paths []string) {
	for _, path := range paths {
		if strings.TrimSpace(path) == "" {
			continue
		}
		_ = os.Remove(filepath.Clean(path))
	}
}

func buildAIUploadDataURL(contentType string, data []byte) string {
	normalized := normalizeAIUploadContentType(contentType, http.DetectContentType(data))
	return "data:" + normalized + ";base64," + base64.StdEncoding.EncodeToString(data)
}

func normalizeAIUploadContentType(primary, fallback string) string {
	primary = strings.TrimSpace(primary)
	if primary != "" && !strings.EqualFold(primary, "application/octet-stream") {
		return primary
	}
	fallback = strings.TrimSpace(fallback)
	if fallback != "" {
		return fallback
	}
	return "application/octet-stream"
}

func detectAIStoredFileKind(name, contentType string) string {
	contentType = strings.ToLower(strings.TrimSpace(contentType))
	if strings.HasPrefix(contentType, "image/") || isAIImageExtension(filepath.Ext(strings.ToLower(name))) {
		return aiUploadKindImage
	}
	return aiUploadKindText
}

func detectAIChatUploadKind(name, contentType string, data []byte) string {
	lowerContentType := strings.ToLower(strings.TrimSpace(contentType))
	lowerExt := strings.ToLower(filepath.Ext(name))
	detectedType := strings.ToLower(strings.TrimSpace(http.DetectContentType(data)))

	if strings.HasPrefix(lowerContentType, "image/") || strings.HasPrefix(detectedType, "image/") || isAIImageExtension(lowerExt) {
		return aiUploadKindImage
	}

	if isAITextContentType(lowerContentType) || isAITextContentType(detectedType) || isAITextExtension(lowerExt) {
		if utf8.Valid(data) {
			return aiUploadKindText
		}
	}
	return ""
}

func isAIImageExtension(ext string) bool {
	switch strings.ToLower(strings.TrimSpace(ext)) {
	case ".png", ".jpg", ".jpeg", ".webp", ".gif", ".bmp":
		return true
	default:
		return false
	}
}

func isAITextExtension(ext string) bool {
	switch strings.ToLower(strings.TrimSpace(ext)) {
	case ".txt", ".log", ".md", ".markdown", ".yaml", ".yml", ".json", ".csv", ".ini", ".conf", ".cfg", ".properties", ".xml", ".html", ".htm", ".sql", ".sh", ".bash", ".zsh", ".py", ".js", ".ts", ".tsx", ".jsx", ".go", ".java", ".rb", ".php":
		return true
	default:
		return false
	}
}

func isAITextContentType(contentType string) bool {
	contentType = strings.ToLower(strings.TrimSpace(contentType))
	if strings.HasPrefix(contentType, "text/") {
		return true
	}
	switch contentType {
	case "application/json", "application/xml", "application/x-yaml", "application/yaml", "application/toml", "application/x-sh":
		return true
	default:
		return false
	}
}

func sanitizeAIUploadName(name string) string {
	name = strings.TrimSpace(filepath.Base(name))
	name = strings.ReplaceAll(name, "\x00", "")
	if name == "." || name == string(filepath.Separator) {
		return ""
	}
	return name
}

func randomHex(n int) (string, error) {
	if n <= 0 {
		return "", nil
	}
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
