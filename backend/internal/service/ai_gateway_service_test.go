package service

import (
	"context"
	"testing"
)

func TestResolveInvocationTarget(t *testing.T) {
	t.Skip("跳过需要数据库的测试")
	tests := []struct {
		name    string
		req     AIGatewayRequest
		wantErr bool
	}{
		{
			name: "valid request with provider and model",
			req: AIGatewayRequest{
				ProviderID: uint64Ptr(1),
				ModelID:    uint64Ptr(1),
			},
			wantErr: false,
		},
		{
			name: "request with prefer model code",
			req: AIGatewayRequest{
				PreferModelCode: "gpt-4",
			},
			wantErr: false,
		},
		{
			name: "request with invalid provider",
			req: AIGatewayRequest{
				ProviderID: uint64Ptr(999),
			},
			wantErr: true,
		},
		{
			name: "request with invalid model",
			req: AIGatewayRequest{
				ModelID: uint64Ptr(999),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &AIGatewayService{}
			_, _, _, err := service.resolveInvocationTarget(context.Background(), tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("resolveInvocationTarget() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestBuildAISystemPromptV2(t *testing.T) {
	tests := []struct {
		name string
		mode string
		want string
	}{
		{
			name: "diagnose mode",
			mode: "diagnose",
			want: "诊断模式提示词",
		},
		{
			name: "chat mode",
			mode: "chat",
			want: "聊天模式提示词",
		},
		{
			name: "default mode",
			mode: "",
			want: "默认提示词",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildAISystemPromptV2(tt.mode)
			if len(got) == 0 {
				t.Errorf("buildAISystemPromptV2() returned empty string")
			}
		})
	}
}

func TestInvoke(t *testing.T) {
	t.Skip("跳过需要数据库的测试")
	tests := []struct {
		name    string
		req     AIGatewayRequest
		wantErr bool
	}{
		{
			name: "mock provider request",
			req: AIGatewayRequest{
				ProviderID: uint64Ptr(1),
				ModelID:    uint64Ptr(1),
			},
			wantErr: false,
		},
		{
			name: "openai compatible provider request",
			req: AIGatewayRequest{
				ProviderID: uint64Ptr(2),
				ModelID:    uint64Ptr(2),
			},
			wantErr: false,
		},
		{
			name: "unsupported provider type",
			req: AIGatewayRequest{
				ProviderID: uint64Ptr(3),
				ModelID:    uint64Ptr(3),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &AIGatewayService{}
			_, err := service.Invoke(context.Background(), tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("Invoke() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func uint64Ptr(v uint64) *uint64 {
	return &v
}
