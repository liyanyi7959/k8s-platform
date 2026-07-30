package application

import (
	"context"
	"errors"
	"testing"

	aidomain "k8s-platform-backend/internal/ai/domain"
)

func TestResolveChatInvocationTargetPrefersPositiveRequestOverrides(t *testing.T) {
	conversationProviderID := uint64(7)
	conversationModelID := uint64(11)
	requestProviderID := uint64(13)
	requestModelID := uint64(17)

	providerID, modelID := ResolveChatInvocationTarget(ChatInvocationTargetInput{
		ConversationProviderID: &conversationProviderID,
		ConversationModelID:    &conversationModelID,
		RequestProviderID:      &requestProviderID,
		RequestModelID:         &requestModelID,
	})
	if providerID != &requestProviderID || modelID != &requestModelID {
		t.Fatalf("resolved target = (%v, %v), want request overrides", providerID, modelID)
	}

	zero := uint64(0)
	providerID, modelID = ResolveChatInvocationTarget(ChatInvocationTargetInput{
		ConversationProviderID: &conversationProviderID,
		ConversationModelID:    &conversationModelID,
		RequestProviderID:      &zero,
		RequestModelID:         &zero,
	})
	if providerID != &conversationProviderID || modelID != &conversationModelID {
		t.Fatalf("zero request overrides should retain conversation target: (%v, %v)", providerID, modelID)
	}
}

func TestBuildChatMessageScopeSnapshot(t *testing.T) {
	providerID := uint64(7)
	modelID := uint64(11)
	snapshot := BuildChatMessageScopeSnapshot(ChatMessageScopeSnapshotInput{
		ClusterID:      5,
		ConversationID: 23,
		Scope:          NewChatRequestScope(" chat ", " payments ", " Deployment ", " api "),
		PreferModel:    " gpt-4.1 ",
		ProviderID:     &providerID,
		ModelID:        &modelID,
		Images: []aidomain.JSONMap{{
			"name":         "image.png",
			"content_type": "image/png",
			"data_url":     "data:image/png;base64,abc",
			"size":         int64(123),
		}},
		Attachments: []AIMessageAttachmentItem{{
			ID:           12,
			OriginalName: "events.log",
			ContentType:  "text/plain",
			FileSize:     256,
			FileKind:     "text",
			DownloadURL:  "/api/v2/ai/files/12/content",
		}},
	})

	requestScope, ok := snapshot["request_scope"].(aidomain.JSONMap)
	if !ok {
		t.Fatalf("request_scope = %#v, want JSONMap", snapshot["request_scope"])
	}
	for key, want := range map[string]any{
		"cluster_id": uint64(5), "conversation_id": uint64(23), "assistant_mode": "chat", "namespace": "payments",
		"resource_kind": "Deployment", "resource_name": "api", "prefer_model": "gpt-4.1", "provider_id": providerID,
		"model_id": modelID, "image_count": 1, "attachment_count": 1,
	} {
		if got := requestScope[key]; got != want {
			t.Fatalf("request_scope[%q] = %#v, want %#v", key, got, want)
		}
	}
	images, ok := snapshot["request_images"].([]aidomain.JSONMap)
	if !ok || len(images) != 1 || images[0]["name"] != "image.png" {
		t.Fatalf("request_images = %#v", snapshot["request_images"])
	}
	attachments, ok := snapshot["request_attachments"].([]aidomain.JSONMap)
	if !ok || len(attachments) != 1 || attachments[0]["original_name"] != "events.log" {
		t.Fatalf("request_attachments = %#v", snapshot["request_attachments"])
	}
}

func TestFilterScopedChatHistoryStopsAtPreviousScope(t *testing.T) {
	rows := []ChatHistoryMessage{
		chatHistoryMessage(1, "user", "show deployment a", "diagnose", "payments", "Deployment", "api-a"),
		chatHistoryMessage(2, "assistant", "deployment a reply", "diagnose", "payments", "Deployment", "api-a"),
		chatHistoryMessage(3, "user", "show deployment b", "diagnose", "payments", "Deployment", "api-b"),
	}

	got := FilterScopedChatHistory(rows, NewChatRequestScope("diagnose", "payments", "Deployment", "api-b"))
	if len(got) != 1 || got[0].ID != 3 {
		t.Fatalf("filtered rows = %#v, want ID 3", got)
	}
}

func TestFilterScopedChatHistoryKeepsCurrentScopeAndSystemMessage(t *testing.T) {
	rows := []ChatHistoryMessage{
		chatHistoryMessage(10, "user", "first question", "chat", "devops", "Deployment", "bkci-auth"),
		{ID: 11, Role: "system", Content: "scope guard"},
		chatHistoryMessage(12, "assistant", "first reply", "chat", "devops", "Deployment", "bkci-auth"),
		chatHistoryMessage(13, "user", "follow up", "chat", "devops", "Deployment", "bkci-auth"),
	}

	got := FilterScopedChatHistory(rows, NewChatRequestScope("chat", "devops", "Deployment", "bkci-auth"))
	if len(got) != 4 {
		t.Fatalf("filtered rows = %#v, want current scoped chain", got)
	}
	for index, wantID := range []uint64{10, 11, 12, 13} {
		if got[index].ID != wantID {
			t.Fatalf("filtered rows[%d].ID = %d, want %d", index, got[index].ID, wantID)
		}
	}
}

func TestFilterScopedChatHistoryRetainsAllForUnscopedTurn(t *testing.T) {
	rows := []ChatHistoryMessage{
		chatHistoryMessage(1, "user", "first", "chat", "devops", "Deployment", "api"),
		chatHistoryMessage(2, "assistant", "reply", "diagnose", "payments", "Deployment", "worker"),
	}
	got := FilterScopedChatHistory(rows, NewChatRequestScope("chat", "", "", ""))
	if len(got) != 2 || got[0].ID != 1 || got[1].ID != 2 {
		t.Fatalf("unscoped history = %#v", got)
	}
}

func TestChatTurnFailurePolicy(t *testing.T) {
	if got := ConversationRunStatusFromError(nil); got != "failed" {
		t.Fatalf("ConversationRunStatusFromError(nil) = %q", got)
	}
	if got := ConversationRunStatusFromError(context.Canceled); got != "cancelled" {
		t.Fatalf("ConversationRunStatusFromError(canceled) = %q", got)
	}
	if got := ConversationRunStatusFromError(context.DeadlineExceeded); got != "failed" {
		t.Fatalf("ConversationRunStatusFromError(timeout) = %q", got)
	}
	if got := MessageStatusFromError(context.Canceled); got != "cancelled" {
		t.Fatalf("MessageStatusFromError(canceled) = %q", got)
	}
	if got := MessageStatusFromError(errors.New("failure")); got != "failed" {
		t.Fatalf("MessageStatusFromError(failure) = %q", got)
	}
	if got := BuildAssistantFailureReply(context.Canceled); got != "本轮回答已取消，平台已停止继续生成结果。你可以调整问题或范围后重新发送。" {
		t.Fatalf("canceled reply = %q", got)
	}
	if got := BuildAssistantFailureReply(context.DeadlineExceeded); got != "本轮回答处理超时，AI 未能在限定时间内完成。建议缩小排查范围后重试。" {
		t.Fatalf("timeout reply = %q", got)
	}
	if got := BuildAssistantFailureReply(errors.New("gateway unavailable")); got != "本轮回答未成功完成。原因：gateway unavailable" {
		t.Fatalf("failure reply = %q", got)
	}
	if got := BuildAssistantFailureReply(ErrorWithMessage(ErrConflict, "provider overloaded")); got != "本轮回答未成功完成。原因：provider overloaded" {
		t.Fatalf("user-facing failure reply = %q", got)
	}
	if got := BuildAssistantFailureReply(errors.New("")); got != "本轮回答未成功完成。原因：本轮回答失败，请稍后重试。" {
		t.Fatalf("empty failure reply = %q", got)
	}
}

func TestUserFacingError(t *testing.T) {
	if got := UserFacingError(nil); got != "" {
		t.Fatalf("UserFacingError(nil) = %q, want empty", got)
	}
	if got := UserFacingError(errors.New("gateway unavailable")); got != "gateway unavailable" {
		t.Fatalf("UserFacingError(plain) = %q", got)
	}
	if got := UserFacingError(ErrorWithMessage(ErrConflict, "provider overloaded")); got != "provider overloaded" {
		t.Fatalf("UserFacingError(application error) = %q", got)
	}
}

func chatHistoryMessage(id uint64, role, content, mode, namespace, kind, name string) ChatHistoryMessage {
	return ChatHistoryMessage{
		ID: id, Role: role, Content: content,
		Structured: aidomain.JSONMap{"request_scope": aidomain.JSONMap{
			"assistant_mode": mode,
			"namespace":      namespace,
			"resource_kind":  kind,
			"resource_name":  name,
		}},
	}
}
