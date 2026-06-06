package email

import (
	"strings"
	"testing"
)

func TestNew_FallsBackToNoop(t *testing.T) {
	if New(Config{}).Enabled() {
		t.Error("unconfigured mailer should be a disabled Noop")
	}
	if !New(Config{Host: "smtp.example.com", From: "a@b.co"}).Enabled() {
		t.Error("configured mailer should be enabled")
	}
}

func TestBuildMessage(t *testing.T) {
	msg := string(BuildMessage("from@x.co", "to@y.co", "Subj", "Hello"))
	for _, want := range []string{"From: from@x.co\r\n", "To: to@y.co\r\n", "Subject: Subj\r\n", "Content-Type: text/plain", "\r\n\r\nHello"} {
		if !strings.Contains(msg, want) {
			t.Errorf("message missing %q:\n%s", want, msg)
		}
	}
}

func TestBuildMessage_NoHeaderInjection(t *testing.T) {
	// CRLF in the recipient (invitee email) and subject (project name) must not
	// inject new header lines.
	msg := string(BuildMessage("from@x.co", "to@y.co\r\nBcc: evil@x.co", "Subj\r\nBcc: evil@x.co", "Body"))
	if strings.Contains(msg, "\r\nBcc:") {
		t.Errorf("CRLF injection produced an extra header:\n%s", msg)
	}
	header := strings.SplitN(msg, "\r\n\r\n", 2)[0]
	if strings.Count(header, "\r\n") != 4 { // From, To, Subject, MIME-Version, Content-Type (4 internal CRLF)
		t.Errorf("unexpected header line count:\n%s", header)
	}
}
