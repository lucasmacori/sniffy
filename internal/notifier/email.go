package notifier

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"net/smtp"
	"strings"
	"time"

	"github.com/jordan-wright/email"
	"github.com/lucasmacori/sniffy/internal/detector"
)

// EmailNotifier sends notifications via email
type EmailNotifier struct {
	BaseNotifier
	host     string
	port     int
	username string
	password string
	from     string
	to       string
}

// NewEmailNotifier creates a new email notifier
func NewEmailNotifier(host string, port int, username, password, from, to string, threshold float64) *EmailNotifier {
	return &EmailNotifier{
		BaseNotifier: BaseNotifier{
			name:      "email",
			threshold: threshold,
		},
		host:     host,
		port:     port,
		username: username,
		password: password,
		from:     from,
		to:       to,
	}
}

// Notify sends an email notification about a finding
func (e *EmailNotifier) Notify(ctx context.Context, finding detector.Finding) error {
	if e.host == "" || e.to == "" {
		return fmt.Errorf("email notifier not configured")
	}

	subject := fmt.Sprintf("[Sniffy] Potential Credential Leak Detected - %s", finding.SecretType)

	htmlBody, err := e.buildHTML(finding)
	if err != nil {
		return fmt.Errorf("build email html failed: %w", err)
	}

	em := email.NewEmail()
	em.From = e.from
	em.To = strings.Split(e.to, ",")
	em.Subject = subject
	em.HTML = []byte(htmlBody)

	addr := fmt.Sprintf("%s:%d", e.host, e.port)
	var auth smtp.Auth
	if e.username != "" && e.password != "" {
		auth = smtp.PlainAuth("", e.username, e.password, e.host)
	}

	if err := em.Send(addr, auth); err != nil {
		return fmt.Errorf("send email failed: %w", err)
	}

	return nil
}

func (e *EmailNotifier) buildHTML(finding detector.Finding) (string, error) {
	tmpl := `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; background-color: #f5f5f5; }
        .container { max-width: 800px; margin: 0 auto; background: white; padding: 30px; border-radius: 8px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        .header { background: #dc3545; color: white; padding: 20px; border-radius: 5px; margin-bottom: 20px; }
        .header h1 { margin: 0; font-size: 24px; }
        .alert { background: #fff3cd; border: 1px solid #ffeaa7; padding: 15px; border-radius: 5px; margin-bottom: 20px; }
        .detail { margin-bottom: 15px; }
        .detail-label { font-weight: bold; color: #555; display: inline-block; width: 150px; }
        .detail-value { font-family: monospace; background: #f8f9fa; padding: 5px 10px; border-radius: 3px; }
        .secret-box { background: #f8d7da; border: 1px solid #f5c6cb; padding: 15px; border-radius: 5px; margin: 20px 0; }
        .secret-value { font-family: monospace; font-size: 14px; word-break: break-all; color: #721c24; }
        .confidence-high { color: #dc3545; font-weight: bold; }
        .confidence-medium { color: #fd7e14; font-weight: bold; }
        .confidence-low { color: #ffc107; font-weight: bold; }
        .footer { margin-top: 30px; padding-top: 20px; border-top: 1px solid #dee2e6; color: #6c757d; font-size: 12px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🚨 Credential Leak Detected</h1>
        </div>
        
        <div class="alert">
            <strong>Warning:</strong> A potential credential leak has been detected in a public GitHub repository. 
            Please review and rotate the exposed credentials immediately.
        </div>

        <div class="detail">
            <span class="detail-label">Repository:</span>
            <span class="detail-value">{{.Repository}}</span>
        </div>

        <div class="detail">
            <span class="detail-label">Secret Type:</span>
            <span class="detail-value">{{.SecretType}}</span>
        </div>

        <div class="detail">
            <span class="detail-label">File Path:</span>
            <span class="detail-value">{{.FilePath}}</span>
        </div>

        <div class="detail">
            <span class="detail-label">Line Number:</span>
            <span class="detail-value">{{.LineNumber}}</span>
        </div>

        {{if .CommitHash}}
        <div class="detail">
            <span class="detail-label">Commit:</span>
            <span class="detail-value">{{.CommitHash}}</span>
        </div>

        <div class="detail">
            <span class="detail-label">Author:</span>
            <span class="detail-value">{{.CommitAuthor}} ({{.CommitEmail}})</span>
        </div>
        {{end}}

        <div class="detail">
            <span class="detail-label">Source:</span>
            <span class="detail-value">{{.Source}}</span>
        </div>

        <div class="detail">
            <span class="detail-label">Confidence:</span>
            <span class="detail-value {{.ConfidenceClass}}">{{.Confidence}}%</span>
        </div>

        <div class="secret-box">
            <strong>Detected Secret:</strong><br>
            <div class="secret-value">{{.SecretValue}}</div>
        </div>

        <div class="footer">
            <p>This alert was generated by Sniffy - Automated Credential Leak Detection</p>
            <p>Time: {{.Timestamp}}</p>
        </div>
    </div>
</body>
</html>`

	type templateData struct {
		detector.Finding
		ConfidenceClass string
		Timestamp       string
	}

	data := templateData{
		Finding:         finding,
		Timestamp:       time.Now().Format("2006-01-02 15:04:05 UTC"),
	}

	if finding.Confidence >= 70 {
		data.ConfidenceClass = "confidence-high"
	} else if finding.Confidence >= 40 {
		data.ConfidenceClass = "confidence-medium"
	} else {
		data.ConfidenceClass = "confidence-low"
	}

	t := template.Must(template.New("email").Parse(tmpl))
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}
