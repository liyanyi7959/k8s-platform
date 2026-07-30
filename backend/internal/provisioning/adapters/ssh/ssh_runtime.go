package ssh

import (
	"context"
	"errors"

	"golang.org/x/crypto/ssh"
	"gorm.io/gorm"

	provisionapp "k8s-platform-backend/internal/provisioning/application"
	model "k8s-platform-backend/internal/provisioning/domain"
	provisionports "k8s-platform-backend/internal/provisioning/ports"
	secretcrypto "k8s-platform-backend/internal/transport/secretcrypto"
	sshtransport "k8s-platform-backend/internal/transport/ssh"
)

// SSHRuntime owns the registered-server SSH probe and terminal connection
// boundary. Deployment-plan execution is owned by DeploymentExecutor and
// reuses the same narrow service transport port.
type SSHRuntime struct {
	db            *gorm.DB
	encryptionKey string
}

func NewSSHRuntime(db *gorm.DB, encryptionKey string) *SSHRuntime {
	return &SSHRuntime{db: db, encryptionKey: encryptionKey}
}

func (r *SSHRuntime) ProbeServerSSH(ctx context.Context, id uint64) (provisionports.SSHProbeResult, error) {
	if id == 0 {
		return provisionports.SSHProbeResult{}, provisionapp.ErrInvalidParams
	}
	server, credential, err := r.resolveServerSSHConfig(ctx, id)
	if err != nil {
		_ = r.updateServerStatus(ctx, id, "unavailable")
		return provisionports.SSHProbeResult{}, err
	}
	probe, err := sshtransport.Probe(ctx, serverSSHConfig(server), credential)
	result := provisionports.SSHProbeResult{
		OS: probe.OS, OSVersion: probe.OSVersion, Kernel: probe.Kernel,
		CPUCores: probe.CPUCores, MemoryMB: probe.MemoryMB, DiskGB: probe.DiskGB,
	}
	status := "available"
	if err != nil {
		status = "unavailable"
		result = provisionports.SSHProbeResult{Status: status, Message: err.Error()}
	} else {
		result.Status = status
		result.Message = "SSH 连接成功"
	}
	updates := map[string]any{"status": status}
	if result.OS != "" {
		updates["os"] = result.OS
	}
	if result.OSVersion != "" {
		updates["os_version"] = result.OSVersion
	}
	if result.Kernel != "" {
		updates["kernel"] = result.Kernel
	}
	if result.CPUCores != nil {
		updates["cpu_cores"] = *result.CPUCores
	}
	if result.MemoryMB != nil {
		updates["memory_mb"] = *result.MemoryMB
	}
	if result.DiskGB != nil {
		updates["disk_gb"] = *result.DiskGB
	}
	if r == nil || r.db == nil {
		return provisionports.SSHProbeResult{}, provisionapp.ErrConflict
	}
	if dbErr := r.db.WithContext(ctx).Model(&model.DeployServer{}).Where("deleted_at IS NULL AND id = ?", id).Updates(updates).Error; dbErr != nil {
		return provisionports.SSHProbeResult{}, dbErr
	}
	if err != nil {
		return result, provisionapp.ErrWithMessage(provisionapp.ErrInvalidParams, result.Message)
	}
	return result, nil
}

// OpenServerSSH creates a terminal client and returns only the target name
// needed for audit logging. Credentials remain inside the runtime.
func (r *SSHRuntime) OpenServerSSH(ctx context.Context, id uint64) (*ssh.Client, string, error) {
	if id == 0 {
		return nil, "", provisionapp.ErrInvalidParams
	}
	server, credential, err := r.resolveServerSSHConfig(ctx, id)
	if err != nil {
		_ = r.updateServerStatus(ctx, id, "unavailable")
		return nil, "", err
	}
	client, err := sshtransport.Dial(ctx, serverSSHConfig(server), credential)
	if err != nil {
		_ = r.updateServerStatus(ctx, id, "unavailable")
		return nil, "", provisionapp.ErrWithMessage(provisionapp.ErrInvalidParams, err.Error())
	}
	_ = r.updateServerStatus(ctx, id, "available")
	return client, server.Name, nil
}

func serverSSHConfig(server model.DeployServer) sshtransport.Config {
	return sshtransport.Config{Host: server.IP, Port: server.SSHPort, User: server.User, AuthType: server.AuthType}
}

func (r *SSHRuntime) resolveServerSSHConfig(ctx context.Context, id uint64) (model.DeployServer, string, error) {
	if r == nil || r.db == nil {
		return model.DeployServer{}, "", provisionapp.ErrConflict
	}
	var server model.DeployServer
	if err := r.db.WithContext(ctx).Where("deleted_at IS NULL AND id = ?", id).First(&server).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.DeployServer{}, "", provisionapp.ErrNotFound
		}
		return model.DeployServer{}, "", err
	}

	credentialEnc := server.CredentialEnc
	if server.CredentialID != nil && *server.CredentialID > 0 {
		var credential model.SSHCredential
		if err := r.db.WithContext(ctx).Where("deleted_at IS NULL AND id = ?", *server.CredentialID).First(&credential).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return model.DeployServer{}, "", provisionapp.ErrWithMessage(provisionapp.ErrNotFound, "关联的凭据不存在")
			}
			return model.DeployServer{}, "", err
		}
		credentialEnc = credential.CredentialEnc
		server.AuthType = credential.AuthType
	}

	credential, err := secretcrypto.Decrypt(r.encryptionKey, credentialEnc)
	if err != nil {
		return model.DeployServer{}, "", provisionapp.ErrWithMessage(nil, "服务器凭证解密失败")
	}
	return server, credential, nil
}

func (r *SSHRuntime) updateServerStatus(ctx context.Context, id uint64, status string) error {
	if r == nil || r.db == nil {
		return provisionapp.ErrConflict
	}
	result := r.db.WithContext(ctx).Model(&model.DeployServer{}).Where("deleted_at IS NULL AND id = ?", id).Update("status", status)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return provisionapp.ErrNotFound
	}
	return nil
}
