// Package mysql implements AI persistence ports using the platform database.
package mysql

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"

	"k8s-platform-backend/internal/ai/domain"
	"k8s-platform-backend/internal/ai/ports"
)

type Repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

func (r *Repository) usable() error {
	if r == nil || r.db == nil {
		return errors.New("db is required")
	}
	return nil
}

func mapNotFound(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ports.ErrNotFound
	}
	return err
}

func (r *Repository) FindConversationInCluster(ctx context.Context, id, clusterID uint64) (domain.AIConversation, error) {
	if err := r.usable(); err != nil {
		return domain.AIConversation{}, err
	}
	var row domain.AIConversation
	err := r.db.WithContext(ctx).Where("deleted_at IS NULL AND id = ? AND cluster_id = ?", id, clusterID).First(&row).Error
	return row, mapNotFound(err)
}

func (r *Repository) CreateActionProposal(ctx context.Context, proposal *domain.AIActionProposal, message domain.AIMessage, changedAt time.Time) error {
	if err := r.usable(); err != nil {
		return err
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(proposal).Error; err != nil {
			return err
		}
		if err := tx.Create(&message).Error; err != nil {
			return err
		}
		return tx.Model(&domain.AIConversation{}).Where("id = ?", proposal.ConversationID).Updates(map[string]any{
			"status": "waiting_confirm", "last_message_at": changedAt,
		}).Error
	})
}

func (r *Repository) FindActionProposalInCluster(ctx context.Context, id, clusterID uint64) (domain.AIActionProposal, error) {
	if err := r.usable(); err != nil {
		return domain.AIActionProposal{}, err
	}
	var row domain.AIActionProposal
	err := r.db.WithContext(ctx).Where("id = ? AND cluster_id = ?", id, clusterID).First(&row).Error
	return row, mapNotFound(err)
}

func (r *Repository) ListActionProposals(ctx context.Context, conversationID uint64) ([]domain.AIActionProposal, error) {
	if err := r.usable(); err != nil {
		return nil, err
	}
	var rows []domain.AIActionProposal
	err := r.db.WithContext(ctx).Where("conversation_id = ?", conversationID).Order("created_at DESC, id DESC").Find(&rows).Error
	return rows, err
}

func (r *Repository) ListActionExecutions(ctx context.Context, proposalIDs []uint64) ([]domain.AIActionExecution, error) {
	if err := r.usable(); err != nil {
		return nil, err
	}
	if len(proposalIDs) == 0 {
		return []domain.AIActionExecution{}, nil
	}
	var rows []domain.AIActionExecution
	err := r.db.WithContext(ctx).Where("proposal_id IN ?", proposalIDs).Order("created_at DESC, id DESC").Find(&rows).Error
	return rows, err
}

func (r *Repository) NextActionExecutionNumber(ctx context.Context, proposalID uint64) (int, error) {
	if err := r.usable(); err != nil {
		return 0, err
	}
	var maxNo int
	err := r.db.WithContext(ctx).Model(&domain.AIActionExecution{}).Select("COALESCE(MAX(execution_no), 0)").Where("proposal_id = ?", proposalID).Scan(&maxNo).Error
	return maxNo + 1, err
}

func (r *Repository) CreateActionExecution(ctx context.Context, execution *domain.AIActionExecution) error {
	if err := r.usable(); err != nil {
		return err
	}
	return r.db.WithContext(ctx).Create(execution).Error
}

func (r *Repository) MarkActionExecuting(ctx context.Context, proposalID uint64, approval ports.ActionApproval) error {
	if err := r.usable(); err != nil {
		return err
	}
	updates := map[string]any{"status": "executing"}
	if approval.SecondApprovedBy != nil {
		updates["second_approved_by"] = *approval.SecondApprovedBy
		updates["second_approved_name"] = approval.SecondApprovedName
		updates["second_approved_at"] = approval.SecondApprovedAt
	} else if approval.ApprovedBy != nil {
		updates["approved_by"] = *approval.ApprovedBy
		updates["approved_by_name"] = approval.ApprovedByName
		updates["approved_at"] = approval.ApprovedAt
	}
	return r.db.WithContext(ctx).Model(&domain.AIActionProposal{}).Where("id = ?", proposalID).Updates(updates).Error
}

func (r *Repository) CompleteActionExecution(ctx context.Context, command ports.ActionCompletion) error {
	if err := r.usable(); err != nil {
		return err
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&domain.AIActionExecution{}).Where("id = ?", command.ExecutionID).Updates(map[string]any{
			"status": "succeeded", "finished_at": &command.FinishedAt, "result_json": command.Result, "error_message": "",
		}).Error; err != nil {
			return err
		}
		if err := tx.Model(&domain.AIActionProposal{}).Where("id = ?", command.ProposalID).Update("status", "succeeded").Error; err != nil {
			return err
		}
		if err := tx.Create(&domain.AIMessage{ConversationID: command.ConversationID, Role: "system", MessageType: "action_execution",
			Content: actionMessage(command.Title, "succeeded", command.Summary), Status: "created", CreatedBy: command.UserID}).Error; err != nil {
			return err
		}
		return tx.Model(&domain.AIConversation{}).Where("id = ?", command.ConversationID).Updates(map[string]any{"status": "open", "last_message_at": &command.FinishedAt}).Error
	})
}

func (r *Repository) FailActionExecution(ctx context.Context, command ports.ActionFailure) error {
	if err := r.usable(); err != nil {
		return err
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		_ = tx.Model(&domain.AIActionExecution{}).Where("id = ?", command.ExecutionID).Updates(map[string]any{
			"status": "failed", "finished_at": &command.FinishedAt, "error_message": command.Message,
		}).Error
		_ = tx.Model(&domain.AIActionProposal{}).Where("id = ?", command.ProposalID).Update("status", "failed").Error
		_ = tx.Create(&domain.AIMessage{ConversationID: command.ConversationID, Role: "system", MessageType: "action_execution",
			Content: actionMessage(command.Title, "failed", command.Message), Status: "created", CreatedBy: command.UserID}).Error
		_ = tx.Model(&domain.AIConversation{}).Where("id = ?", command.ConversationID).Updates(map[string]any{"status": "open", "last_message_at": &command.FinishedAt}).Error
		return nil
	})
}

func actionMessage(title, status, summary string) string {
	return "变更提案执行" + actionStatusLabel(status) + "：" + strings.TrimSpace(title) + "\n" + strings.TrimSpace(summary)
}
func actionStatusLabel(status string) string {
	if status == "succeeded" {
		return "成功"
	}
	return "失败"
}

func (r *Repository) ListConversations(ctx context.Context, filter ports.ConversationFilter) ([]domain.AIConversation, int, error) {
	if err := r.usable(); err != nil {
		return nil, 0, err
	}
	q := r.db.WithContext(ctx).Model(&domain.AIConversation{}).Where("deleted_at IS NULL")
	if filter.ClusterID > 0 {
		q = q.Where("cluster_id = ?", filter.ClusterID)
	}
	if filter.Status != "" {
		q = q.Where("status = ?", filter.Status)
	}
	if filter.AssistantMode != "" {
		q = q.Where("assistant_mode = ?", filter.AssistantMode)
	}
	if filter.Keyword != "" {
		q = q.Where("title LIKE ? OR summary LIKE ?", "%"+filter.Keyword+"%", "%"+filter.Keyword+"%")
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []domain.AIConversation
	if err := q.Order("updated_at DESC, id DESC").Offset((filter.Page - 1) * filter.PageSize).Limit(filter.PageSize).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, int(total), nil
}

func (r *Repository) CountMessages(ctx context.Context, conversationIDs []uint64) (map[uint64]int, error) {
	if err := r.usable(); err != nil {
		return nil, err
	}
	counts := make(map[uint64]int, len(conversationIDs))
	if len(conversationIDs) == 0 {
		return counts, nil
	}
	type row struct {
		ConversationID uint64
		Count          int
	}
	var result []row
	if err := r.db.WithContext(ctx).Model(&domain.AIMessage{}).Select("conversation_id, COUNT(*) AS count").Where("conversation_id IN ?", conversationIDs).Group("conversation_id").Scan(&result).Error; err != nil {
		return nil, err
	}
	for _, item := range result {
		counts[item.ConversationID] = item.Count
	}
	return counts, nil
}

func (r *Repository) CreateConversation(ctx context.Context, command ports.ConversationCreate) (uint64, error) {
	if err := r.usable(); err != nil {
		return 0, err
	}
	conversation := command.Conversation
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var cluster clusterReference
		if err := tx.Select("id").Where("deleted_at IS NULL AND id = ?", conversation.ClusterID).First(&cluster).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ports.ErrClusterNotFound
			}
			return err
		}
		if conversation.ProviderID != nil {
			var provider domain.AIProvider
			if err := tx.Select("id").Where("deleted_at IS NULL AND id = ?", *conversation.ProviderID).First(&provider).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return ports.ErrProviderNotFound
				}
				return err
			}
		}
		if conversation.ModelID != nil {
			var model domain.AIModel
			if err := tx.Where("deleted_at IS NULL AND id = ?", *conversation.ModelID).First(&model).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return ports.ErrModelNotFound
				}
				return err
			}
			if conversation.ProviderID != nil && model.ProviderID != *conversation.ProviderID {
				return ports.ErrModelProviderMismatch
			}
		}
		if err := tx.Create(&conversation).Error; err != nil {
			return err
		}
		if command.OpeningMessage == "" {
			return nil
		}
		return tx.Create(&domain.AIMessage{ConversationID: conversation.ID, Role: "user", MessageType: "text", Content: command.OpeningMessage, Status: "created", CreatedBy: command.UserID}).Error
	})
	return conversation.ID, err
}

func (r *Repository) DeleteConversation(ctx context.Context, id uint64, deletedAt time.Time) error {
	if err := r.usable(); err != nil {
		return err
	}
	result := r.db.WithContext(ctx).Model(&domain.AIConversation{}).Where("deleted_at IS NULL AND id = ?", id).Update("deleted_at", &deletedAt)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ports.ErrNotFound
	}
	return nil
}

func (r *Repository) FindConversation(ctx context.Context, id uint64) (domain.AIConversation, error) {
	if err := r.usable(); err != nil {
		return domain.AIConversation{}, err
	}
	var row domain.AIConversation
	err := r.db.WithContext(ctx).Where("deleted_at IS NULL AND id = ?", id).First(&row).Error
	return row, mapNotFound(err)
}

func (r *Repository) ListConversationMessages(ctx context.Context, conversationID uint64) ([]domain.AIMessage, error) {
	if err := r.usable(); err != nil {
		return nil, err
	}
	var rows []domain.AIMessage
	err := r.db.WithContext(ctx).Where("conversation_id = ?", conversationID).Order("created_at ASC, id ASC").Find(&rows).Error
	return rows, err
}

func (r *Repository) CreateUploadedFiles(ctx context.Context, rows []domain.AIUploadedFile) error {
	if err := r.usable(); err != nil {
		return err
	}
	return r.db.WithContext(ctx).Create(&rows).Error
}
func (r *Repository) FindUploadedFile(ctx context.Context, id uint64) (domain.AIUploadedFile, error) {
	if err := r.usable(); err != nil {
		return domain.AIUploadedFile{}, err
	}
	var row domain.AIUploadedFile
	err := r.db.WithContext(ctx).Where("deleted_at IS NULL AND id = ?", id).First(&row).Error
	return row, mapNotFound(err)
}
func (r *Repository) ListConversationFiles(ctx context.Context, conversationID uint64) ([]domain.AIUploadedFile, error) {
	if err := r.usable(); err != nil {
		return nil, err
	}
	var rows []domain.AIUploadedFile
	err := r.db.WithContext(ctx).Where("deleted_at IS NULL AND conversation_id = ?", conversationID).Order("created_at ASC, id ASC").Find(&rows).Error
	return rows, err
}

func (r *Repository) ListProviders(ctx context.Context) ([]domain.AIProvider, error) {
	if err := r.usable(); err != nil {
		return nil, err
	}
	var rows []domain.AIProvider
	err := r.db.WithContext(ctx).Where("deleted_at IS NULL").Order("priority ASC, id DESC").Find(&rows).Error
	return rows, err
}
func (r *Repository) CreateProvider(ctx context.Context, row *domain.AIProvider) error {
	if err := r.usable(); err != nil {
		return err
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing domain.AIProvider
		err := tx.Where("deleted_at IS NULL AND name = ?", row.Name).First(&existing).Error
		if err == nil {
			return ports.ErrConflict
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		return tx.Create(row).Error
	})
}
func (r *Repository) PatchProvider(ctx context.Context, id uint64, patch ports.ProviderPatch) error {
	if err := r.usable(); err != nil {
		return err
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row domain.AIProvider
		if err := tx.Where("deleted_at IS NULL AND id = ?", id).First(&row).Error; err != nil {
			return mapNotFound(err)
		}
		if patch.Name != nil {
			var existing domain.AIProvider
			err := tx.Select("id").Where("deleted_at IS NULL AND name = ? AND id <> ?", *patch.Name, id).First(&existing).Error
			if err == nil {
				return ports.ErrConflict
			}
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
		}
		updates := providerUpdates(patch)
		if len(updates) == 0 {
			return nil
		}
		return tx.Model(&domain.AIProvider{}).Where("id = ? AND deleted_at IS NULL", id).Updates(updates).Error
	})
}
func providerUpdates(p ports.ProviderPatch) map[string]any {
	u := map[string]any{}
	if p.Name != nil {
		u["name"] = *p.Name
	}
	if p.ProviderType != nil {
		u["provider_type"] = *p.ProviderType
	}
	if p.VendorCode != nil {
		u["vendor_code"] = *p.VendorCode
	}
	if p.BaseURL != nil {
		u["base_url"] = *p.BaseURL
	}
	if p.AuthScheme != nil {
		u["auth_scheme"] = *p.AuthScheme
	}
	if p.APIKeyEnc != nil {
		if *p.APIKeyEnc == nil {
			u["api_key_enc"] = nil
		} else {
			u["api_key_enc"] = **p.APIKeyEnc
		}
	}
	if p.Enabled != nil {
		u["enabled"] = *p.Enabled
	}
	if p.Priority != nil {
		u["priority"] = *p.Priority
	}
	if p.Meta != nil {
		u["meta_json"] = *p.Meta
	}
	return u
}
func (r *Repository) DeleteProvider(ctx context.Context, id uint64, deletedAt time.Time) error {
	if err := r.usable(); err != nil {
		return err
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row domain.AIProvider
		if err := tx.Where("deleted_at IS NULL AND id = ?", id).First(&row).Error; err != nil {
			return mapNotFound(err)
		}
		var count int64
		if err := tx.Model(&domain.AIModel{}).Where("deleted_at IS NULL AND provider_id = ?", id).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return ports.ErrProviderInUse
		}
		return tx.Model(&domain.AIProvider{}).Where("id = ? AND deleted_at IS NULL", id).Updates(map[string]any{"deleted_at": deletedAt, "updated_at": deletedAt}).Error
	})
}
func (r *Repository) ListModels(ctx context.Context, filter ports.ModelFilter) ([]ports.ModelWithProviderName, error) {
	if err := r.usable(); err != nil {
		return nil, err
	}
	type row struct {
		domain.AIModel
		ProviderName string `gorm:"column:provider_name"`
	}
	var rows []row
	q := r.db.WithContext(ctx).Table("ai_models AS m").Select("m.*, p.name AS provider_name").Joins("JOIN ai_providers AS p ON p.id = m.provider_id AND p.deleted_at IS NULL").Where("m.deleted_at IS NULL")
	if filter.ProviderID > 0 {
		q = q.Where("m.provider_id = ?", filter.ProviderID)
	}
	if filter.ModelType != "" {
		q = q.Where("m.model_type = ?", filter.ModelType)
	}
	if filter.Enabled != nil {
		q = q.Where("m.enabled = ?", *filter.Enabled)
	}
	if err := q.Order("m.id DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]ports.ModelWithProviderName, 0, len(rows))
	for _, item := range rows {
		out = append(out, ports.ModelWithProviderName{Model: item.AIModel, ProviderName: item.ProviderName})
	}
	return out, nil
}
func (r *Repository) CreateModel(ctx context.Context, row *domain.AIModel) error {
	if err := r.usable(); err != nil {
		return err
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var p domain.AIProvider
		if err := tx.Select("id").Where("deleted_at IS NULL AND id = ?", row.ProviderID).First(&p).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ports.ErrProviderNotFound
			}
			return err
		}
		var existing domain.AIModel
		err := tx.Where("deleted_at IS NULL AND provider_id = ? AND model_code = ?", row.ProviderID, row.ModelCode).First(&existing).Error
		if err == nil {
			return ports.ErrConflict
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		return tx.Create(row).Error
	})
}
func (r *Repository) PatchModel(ctx context.Context, id uint64, patch ports.ModelPatch) error {
	if err := r.usable(); err != nil {
		return err
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row domain.AIModel
		if err := tx.Where("deleted_at IS NULL AND id = ?", id).First(&row).Error; err != nil {
			return mapNotFound(err)
		}
		nextProviderID := row.ProviderID
		if patch.ProviderID != nil {
			var p domain.AIProvider
			if err := tx.Select("id").Where("deleted_at IS NULL AND id = ?", *patch.ProviderID).First(&p).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return ports.ErrProviderNotFound
				}
				return err
			}
			nextProviderID = *patch.ProviderID
		}
		nextCode := row.ModelCode
		if patch.ModelCode != nil {
			nextCode = *patch.ModelCode
		}
		if nextProviderID != row.ProviderID || nextCode != row.ModelCode {
			var existing domain.AIModel
			err := tx.Select("id").Where("deleted_at IS NULL AND provider_id = ? AND model_code = ? AND id <> ?", nextProviderID, nextCode, id).First(&existing).Error
			if err == nil {
				return ports.ErrConflict
			}
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
		}
		updates := modelUpdates(patch)
		if len(updates) == 0 {
			return nil
		}
		return tx.Model(&domain.AIModel{}).Where("id = ? AND deleted_at IS NULL", id).Updates(updates).Error
	})
}
func modelUpdates(p ports.ModelPatch) map[string]any {
	u := map[string]any{}
	if p.ProviderID != nil {
		u["provider_id"] = *p.ProviderID
	}
	if p.Name != nil {
		u["name"] = *p.Name
	}
	if p.ModelCode != nil {
		u["model_code"] = *p.ModelCode
	}
	if p.ModelType != nil {
		u["model_type"] = *p.ModelType
	}
	if p.Enabled != nil {
		u["enabled"] = *p.Enabled
	}
	if p.SupportsTools != nil {
		u["supports_tools"] = *p.SupportsTools
	}
	if p.SupportsVision != nil {
		u["supports_vision"] = *p.SupportsVision
	}
	if p.SupportsStreaming != nil {
		u["supports_streaming"] = *p.SupportsStreaming
	}
	if p.SupportsReasoning != nil {
		u["supports_reasoning"] = *p.SupportsReasoning
	}
	if p.SupportsStructuredOutput != nil {
		u["supports_structured_output"] = *p.SupportsStructuredOutput
	}
	if p.SupportsImageGeneration != nil {
		u["supports_image_generation"] = *p.SupportsImageGeneration
	}
	if p.SupportsFileInput != nil {
		u["supports_file_input"] = *p.SupportsFileInput
	}
	if p.MaxInputTokens != nil {
		u["max_input_tokens"] = *p.MaxInputTokens
	}
	if p.MaxOutputTokens != nil {
		u["max_output_tokens"] = *p.MaxOutputTokens
	}
	if p.ContextWindow != nil {
		u["context_window"] = *p.ContextWindow
	}
	if p.Meta != nil {
		u["meta_json"] = *p.Meta
	}
	return u
}
func (r *Repository) DeleteModel(ctx context.Context, id uint64, deletedAt time.Time) error {
	if err := r.usable(); err != nil {
		return err
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row domain.AIModel
		if err := tx.Where("deleted_at IS NULL AND id = ?", id).First(&row).Error; err != nil {
			return mapNotFound(err)
		}
		return tx.Model(&domain.AIModel{}).Where("id = ? AND deleted_at IS NULL", id).Updates(map[string]any{"deleted_at": deletedAt, "updated_at": deletedAt}).Error
	})
}

func (r *Repository) GetOrCreateRouteSettings(ctx context.Context) (domain.AIRouteSetting, error) {
	if err := r.usable(); err != nil {
		return domain.AIRouteSetting{}, err
	}
	var row domain.AIRouteSetting
	if err := r.db.WithContext(ctx).First(&row, 1).Error; err == nil {
		return row, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return domain.AIRouteSetting{}, err
	}
	row = domain.AIRouteSetting{ID: 1, RoutingStrategy: "priority_first", AllowFallback: true}
	if err := r.db.WithContext(ctx).Create(&row).Error; err != nil {
		return domain.AIRouteSetting{}, err
	}
	return row, nil
}
func (r *Repository) UpdateRouteSettings(ctx context.Context, id uint64, patch ports.RouteSettingsPatch) error {
	if err := r.usable(); err != nil {
		return err
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, modelID := range []*uint64{patch.DefaultChatModelID, patch.DefaultDiagnoseModelID, patch.DefaultVisionModelID, patch.DefaultImageGenerationModelID} {
			if modelID == nil || *modelID == 0 {
				continue
			}
			var row domain.AIModel
			if err := tx.Select("id").Where("deleted_at IS NULL AND id = ?", *modelID).First(&row).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return ports.ErrModelNotFound
				}
				return err
			}
		}
		if patch.DefaultFallbackProviderID != nil && *patch.DefaultFallbackProviderID != 0 {
			var row domain.AIProvider
			if err := tx.Select("id").Where("deleted_at IS NULL AND id = ?", *patch.DefaultFallbackProviderID).First(&row).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return ports.ErrProviderNotFound
				}
				return err
			}
		}
		u := map[string]any{"routing_strategy": patch.RoutingStrategy}
		if patch.AllowFallback != nil {
			u["allow_fallback"] = *patch.AllowFallback
		}
		if patch.Meta != nil {
			u["meta_json"] = *patch.Meta
		}
		if patch.DefaultChatModelID != nil {
			u["default_chat_model_id"] = nullable(*patch.DefaultChatModelID)
		}
		if patch.DefaultDiagnoseModelID != nil {
			u["default_diagnose_model_id"] = nullable(*patch.DefaultDiagnoseModelID)
		}
		if patch.DefaultVisionModelID != nil {
			u["default_vision_model_id"] = nullable(*patch.DefaultVisionModelID)
		}
		if patch.DefaultImageGenerationModelID != nil {
			u["default_image_generation_model_id"] = nullable(*patch.DefaultImageGenerationModelID)
		}
		if patch.DefaultFallbackProviderID != nil {
			u["default_fallback_provider_id"] = nullable(*patch.DefaultFallbackProviderID)
		}
		return tx.Model(&domain.AIRouteSetting{}).Where("id = ?", id).Updates(u).Error
	})
}
func nullable(value uint64) any {
	if value == 0 {
		return nil
	}
	return value
}

func (r *Repository) ListToolCalls(ctx context.Context, conversationID uint64) ([]domain.AIToolCall, error) {
	if err := r.usable(); err != nil {
		return nil, err
	}
	var rows []domain.AIToolCall
	err := r.db.WithContext(ctx).Where("conversation_id = ?", conversationID).Order("created_at ASC, id ASC").Find(&rows).Error
	return rows, err
}
func (r *Repository) CreateToolCall(ctx context.Context, row *domain.AIToolCall) error {
	if err := r.usable(); err != nil {
		return err
	}
	return r.db.WithContext(ctx).Create(row).Error
}
func (r *Repository) UpdateToolCall(ctx context.Context, id uint64, update ports.ToolCallUpdate) error {
	if err := r.usable(); err != nil {
		return err
	}
	return r.db.WithContext(ctx).Model(&domain.AIToolCall{}).Where("id = ?", id).Updates(map[string]any{"status": update.Status, "result_summary": update.ResultSummary, "result_json": update.Result, "error_message": update.ErrorMessage}).Error
}

type clusterReference struct{ ID uint64 }

func (clusterReference) TableName() string { return "clusters" }
