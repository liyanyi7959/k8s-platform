package service

import (
	"errors"
	"testing"
)

// ────────── ServiceError 基本行为 ──────────

func TestServiceError_Error(t *testing.T) {
	se := &ServiceError{Kind: ErrNotFound, Message: "用户不存在"}
	if se.Error() != "用户不存在" {
		t.Errorf("Error() = %q, want %q", se.Error(), "用户不存在")
	}
}

func TestServiceError_Unwrap(t *testing.T) {
	se := &ServiceError{Kind: ErrConflict, Message: "名称重复"}
	if se.Unwrap() != ErrConflict {
		t.Errorf("Unwrap() = %v, want ErrConflict", se.Unwrap())
	}
}

// ────────── errors.Is 链式判断 ──────────

func TestServiceError_ErrorsIs(t *testing.T) {
	err := ErrWithMessage(ErrNotFound, "记录不存在")
	if !errors.Is(err, ErrNotFound) {
		t.Error("errors.Is should match ErrNotFound")
	}
	if errors.Is(err, ErrConflict) {
		t.Error("errors.Is should NOT match ErrConflict")
	}
}

func TestServiceError_ErrorsAs(t *testing.T) {
	err := ErrWithMessage(ErrInvalidParams, "缺少 name 字段")
	var se *ServiceError
	if !errors.As(err, &se) {
		t.Fatal("errors.As should succeed")
	}
	if se.Message != "缺少 name 字段" {
		t.Errorf("Message = %q, want %q", se.Message, "缺少 name 字段")
	}
}

// ────────── ErrWithMessage 边界 ──────────

func TestErrWithMessage_NilKind(t *testing.T) {
	err := ErrWithMessage(nil, "  some message  ")
	if err == nil {
		t.Fatal("should not be nil")
	}
	if err.Error() != "some message" {
		t.Errorf("Error() = %q, want %q", err.Error(), "some message")
	}
}

func TestErrWithMessage_EmptyMessage(t *testing.T) {
	err := ErrWithMessage(ErrCrypto, "   ")
	if err != ErrCrypto {
		t.Errorf("expected raw sentinel, got %v", err)
	}
}

func TestErrWithMessage_Normal(t *testing.T) {
	err := ErrWithMessage(ErrK8sNetwork, "无法连接 10.0.0.1:6443")
	if !errors.Is(err, ErrK8sNetwork) {
		t.Error("errors.Is should match ErrK8sNetwork")
	}
}

// ────────── UserMessage ──────────

func TestUserMessage_WithServiceError(t *testing.T) {
	err := ErrWithMessage(ErrSSHAuth, "密钥格式错误")
	msg, ok := UserMessage(err)
	if !ok {
		t.Fatal("UserMessage should return ok=true")
	}
	if msg != "密钥格式错误" {
		t.Errorf("msg = %q, want %q", msg, "密钥格式错误")
	}
}

func TestUserMessage_PlainError(t *testing.T) {
	_, ok := UserMessage(ErrNotFound)
	if ok {
		t.Error("UserMessage should return ok=false for plain sentinel")
	}
}

func TestUserMessage_NilError(t *testing.T) {
	_, ok := UserMessage(nil)
	if ok {
		t.Error("UserMessage should return ok=false for nil")
	}
}

// ────────── 哨兵错误独立性 ──────────

func TestSentinels_Independent(t *testing.T) {
	// 确保不同哨兵之间互不匹配
	sentinels := []error{
		ErrNotFound, ErrConflict, ErrInvalidParams, ErrCrypto,
		ErrK8s, ErrK8sNetwork, ErrK8sTimeout, ErrK8sUnauthorized, ErrK8sForbidden, ErrK8sTLS,
		ErrSSHNetwork, ErrSSHTimeout, ErrSSHAuth,
		ErrTaskNotFound, ErrTaskCannotCancel,
	}
	for i := 0; i < len(sentinels); i++ {
		for j := i + 1; j < len(sentinels); j++ {
			if errors.Is(sentinels[i], sentinels[j]) {
				t.Errorf("sentinel[%d] should not match sentinel[%d]", i, j)
			}
		}
	}
}

// ────────── PageResult ──────────

func TestPageResult_Zero(t *testing.T) {
	pr := PageResult[string]{List: nil, Total: 0, Page: 1, PageSize: 10}
	if pr.Total != 0 || pr.Page != 1 || pr.PageSize != 10 {
		t.Error("PageResult zero value mismatch")
	}
	if pr.List != nil {
		t.Error("List should be nil")
	}
}
