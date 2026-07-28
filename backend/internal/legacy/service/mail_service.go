package service

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net/smtp"
	"strings"

	"k8s-platform-backend/internal/config"
)

// ── 邮件发送 Service ──
// 基于 net/smtp 标准库实现，支持 STARTTLS 和 SSL/TLS

type MailService struct {
	cfg config.MailConfig
}

func NewMailService(cfg config.MailConfig) *MailService {
	return &MailService{cfg: cfg}
}

// Enabled 返回邮件服务是否可用
func (ms *MailService) Enabled() bool {
	return ms != nil && ms.cfg.Enabled && ms.cfg.Host != "" && ms.cfg.Username != ""
}

// SendMail 发送纯文本邮件
func (ms *MailService) SendMail(to []string, subject, body string) error {
	if ms == nil || !ms.Enabled() {
		return errors.New("邮件服务未启用")
	}
	if len(to) == 0 {
		return errors.New("收件人不能为空")
	}

	from := ms.cfg.From
	if from == "" {
		from = ms.cfg.Username
	}

	// 构建邮件内容（RFC 822 格式）
	msg := strings.Join([]string{
		fmt.Sprintf("From: %s", from),
		fmt.Sprintf("To: %s", strings.Join(to, ", ")),
		fmt.Sprintf("Subject: %s", subject),
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"Content-Transfer-Encoding: 8bit",
		"",
		body,
	}, "\r\n")

	addr := fmt.Sprintf("%s:%d", ms.cfg.Host, ms.cfg.Port)
	auth := smtp.PlainAuth("", ms.cfg.Username, ms.cfg.Password, ms.cfg.Host)

	// SSL/TLS 模式（通常端口 465）
	if ms.cfg.SSL {
		return ms.sendWithSSL(addr, auth, from, to, msg)
	}

	// STARTTLS 模式（通常端口 587）
	if ms.cfg.StartTLS {
		return ms.sendWithSTARTTLS(addr, auth, from, to, msg)
	}

	// 明文模式（不推荐）
	return smtp.SendMail(addr, auth, from, to, []byte(msg))
}

// sendWithSTARTTLS 使用 STARTTLS 发送邮件
func (ms *MailService) sendWithSTARTTLS(addr string, auth smtp.Auth, from string, to []string, msg string) error {
	tlsConfig := &tls.Config{ServerName: ms.cfg.Host}

	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return fmt.Errorf("TLS 连接失败: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, ms.cfg.Host)
	if err != nil {
		return fmt.Errorf("创建 SMTP 客户端失败: %w", err)
	}
	defer client.Quit()

	if err = client.Auth(auth); err != nil {
		return fmt.Errorf("SMTP 认证失败: %w", err)
	}
	if err = client.Mail(from); err != nil {
		return fmt.Errorf("设置发件人失败: %w", err)
	}
	for _, t := range to {
		if err = client.Rcpt(t); err != nil {
			return fmt.Errorf("设置收件人失败: %w", err)
		}
	}
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("打开数据流失败: %w", err)
	}
	if _, err = fmt.Fprint(w, msg); err != nil {
		return fmt.Errorf("写入邮件内容失败: %w", err)
	}
	if err = w.Close(); err != nil {
		return fmt.Errorf("关闭数据流失败: %w", err)
	}
	return nil
}

// sendWithSSL 使用 SSL/TLS 发送邮件
func (ms *MailService) sendWithSSL(addr string, auth smtp.Auth, from string, to []string, msg string) error {
	tlsConfig := &tls.Config{ServerName: ms.cfg.Host}

	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return fmt.Errorf("SSL 连接失败: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, ms.cfg.Host)
	if err != nil {
		return fmt.Errorf("创建 SMTP 客户端失败: %w", err)
	}
	defer client.Quit()

	if err = client.Auth(auth); err != nil {
		return fmt.Errorf("SMTP 认证失败: %w", err)
	}
	if err = client.Mail(from); err != nil {
		return fmt.Errorf("设置发件人失败: %w", err)
	}
	for _, t := range to {
		if err = client.Rcpt(t); err != nil {
			return fmt.Errorf("设置收件人失败: %w", err)
		}
	}
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("打开数据流失败: %w", err)
	}
	if _, err = fmt.Fprint(w, msg); err != nil {
		return fmt.Errorf("写入邮件内容失败: %w", err)
	}
	if err = w.Close(); err != nil {
		return fmt.Errorf("关闭数据流失败: %w", err)
	}
	return nil
}
