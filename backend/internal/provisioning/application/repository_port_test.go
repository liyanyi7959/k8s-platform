package application

import (
	"context"
	"errors"
	"testing"

	provisiondomain "k8s-platform-backend/internal/provisioning/domain"
	"k8s-platform-backend/internal/provisioning/ports"
)

// memoryProvisioningRepository embeds the complete port but implements only
// the methods exercised by these use-case tests. Unused port methods remain
// unavailable, which keeps tests focused on application behavior rather than
// a database implementation.
type memoryProvisioningRepository struct {
	ports.Repository
	templates map[string]provisiondomain.AppTemplate
	servers   []provisiondomain.DeployServer
}

func (r *memoryProvisioningRepository) Transaction(_ context.Context, fn func(ports.Repository) error) error {
	return fn(r)
}
func (r *memoryProvisioningRepository) AppTemplateNameExists(_ context.Context, name string, excludeID uint64) (bool, error) {
	template, ok := r.templates[name]
	return ok && template.ID != excludeID, nil
}
func (r *memoryProvisioningRepository) CreateAppTemplate(_ context.Context, template *provisiondomain.AppTemplate) error {
	template.ID = uint64(len(r.templates) + 1)
	r.templates[template.Name] = *template
	return nil
}
func (r *memoryProvisioningRepository) ServerExists(_ context.Context, ip string, port int, excludeID uint64) (bool, error) {
	for _, server := range r.servers {
		if server.ID != excludeID && server.IP == ip && server.SSHPort == port {
			return true, nil
		}
	}
	return false, nil
}
func (r *memoryProvisioningRepository) CreateServer(_ context.Context, server *provisiondomain.DeployServer) error {
	server.ID = uint64(len(r.servers) + 1)
	r.servers = append(r.servers, *server)
	return nil
}

func TestAppTemplateServiceUsesRepositoryPort(t *testing.T) {
	repository := &memoryProvisioningRepository{templates: map[string]provisiondomain.AppTemplate{}}
	service := NewAppTemplateService(repository)
	template := &provisiondomain.AppTemplate{Name: " nginx "}
	if err := service.CreateAppTemplate(context.Background(), template); err != nil {
		t.Fatalf("CreateAppTemplate() error = %v", err)
	}
	if template.ID == 0 || template.Name != "nginx" {
		t.Fatalf("created template = %#v", template)
	}
	if err := service.CreateAppTemplate(context.Background(), &provisiondomain.AppTemplate{Name: "nginx"}); err != ErrConflict {
		t.Fatalf("duplicate error = %v, want %v", err, ErrConflict)
	}
}

func TestServerServiceUsesRepositoryTransactionPort(t *testing.T) {
	repository := &memoryProvisioningRepository{}
	service := NewServerService(repository, "test-key")
	id, err := service.Create(context.Background(), CreateDeployServerRequest{Name: "node-a", IP: "10.0.0.1", Credential: "secret"})
	if err != nil || id == 0 {
		t.Fatalf("Create() = (%d, %v)", id, err)
	}
	_, err = service.Create(context.Background(), CreateDeployServerRequest{Name: "node-b", IP: "10.0.0.1", Credential: "secret"})
	if err == nil || !errors.Is(err, ErrConflict) {
		t.Fatalf("duplicate Create() error = %v", err)
	}
}
