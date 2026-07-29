package domain

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

var (
	ErrNotFound            = errors.New("cluster not found")
	ErrConflict            = errors.New("cluster conflict")
	ErrValidation          = errors.New("cluster validation failed")
	ErrCrypto              = errors.New("cluster credential crypto failed")
	ErrRuntimeUnauthorized = errors.New("cluster credential unauthorized")
	ErrRuntimeForbidden    = errors.New("cluster access forbidden")
	ErrRuntimeNetwork      = errors.New("cluster network failure")
	ErrRuntimeTimeout      = errors.New("cluster timeout")
	ErrRuntimeTLS          = errors.New("cluster tls failure")
	ErrRuntime             = errors.New("cluster runtime failure")
)

var namePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]*$`)

type Cluster struct {
	ID                   uint64
	Name                 string
	Type                 string
	Status               string
	KubeconfigEncrypted  *string
	K8sVersion           string
	Description          string
	NodeCount            int
	LastHealthAt         *time.Time
	MonitorSource        string
	PrometheusURL        string
	PrometheusStatus     string
	PrometheusDetectedAt *time.Time
	CreatedAt            time.Time
	UpdatedAt            time.Time
	DeletedAt            *time.Time
}

func NewImported(name, kubeconfig, description string) (Cluster, string, error) {
	name, kubeconfig, description = strings.TrimSpace(name), strings.TrimSpace(kubeconfig), strings.TrimSpace(description)
	if err := ValidateName(name); err != nil {
		return Cluster{}, "", err
	}
	if err := ValidateKubeconfig(kubeconfig); err != nil {
		return Cluster{}, "", err
	}
	if utf8.RuneCountInString(description) > 500 {
		return Cluster{}, "", fmt.Errorf("%w: 备注最长 500 字符", ErrValidation)
	}
	return Cluster{Name: name, Type: "imported", Status: "active", Description: description}, kubeconfig, nil
}

func ValidateName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("%w: 集群名称不能为空", ErrValidation)
	}
	if len(name) > 120 || !namePattern.MatchString(name) {
		return fmt.Errorf("%w: 集群名称需以字母或数字开头，最长 120 字符，仅可包含字母、数字、点、横线和下划线", ErrValidation)
	}
	return nil
}

func ValidateKubeconfig(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return fmt.Errorf("%w: kubeconfig 不能为空", ErrValidation)
	}
	if len(value) > 1024*1024 {
		return fmt.Errorf("%w: kubeconfig 内容不能超过 1MB", ErrValidation)
	}
	return nil
}
