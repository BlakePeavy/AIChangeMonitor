package redact

import (
	"regexp"
	"strings"
	"unicode"
)

var (
	reAWSKey          = regexp.MustCompile(`AKIA[0-9A-Z]{16}`)
	reAWSSecretAssign = regexp.MustCompile(`(?i)(aws_secret_access_key|aws_secret_key)\s*[:=]\s*\S+`)
	rePEM             = regexp.MustCompile(`-----BEGIN [A-Z0-9 ]{0,40}PRIVATE KEY-----[\s\S]*?-----END [A-Z0-9 ]{0,40}PRIVATE KEY-----`)
	rePEMOneline      = regexp.MustCompile(`-----BEGIN [A-Z0-9 ]{0,40}PRIVATE KEY-----`)
	reGitHubPAT       = regexp.MustCompile(`ghp_[A-Za-z0-9_]{20,}`)
	reGitHubFine      = regexp.MustCompile(`github_pat_[A-Za-z0-9_]{20,}`)
	reOpenAI          = regexp.MustCompile(`sk-[A-Za-z0-9_-]{20,}`)
	reAnthropic       = regexp.MustCompile(`sk-ant-[A-Za-z0-9_-]{20,}`)
	reSlack           = regexp.MustCompile(`xox[baprs]-[A-Za-z0-9-]{10,}`)
	reJWT             = regexp.MustCompile(`eyJ[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}`)
	reEnvAssign       = regexp.MustCompile(`(?i)\b([A-Z][A-Z0-9_]{1,64}(SECRET|TOKEN|PASSWORD|PASSWD|API[_-]?KEY|AUTH|CREDENTIAL|PRIVATE[_-]?KEY|ACCESS[_-]?KEY)[A-Z0-9_]*)\s*[:=]\s*([^\s#]+)`)
	reGenericAssign   = regexp.MustCompile(`(?i)\b(password|secret|token|api[_-]?key|authorization)\s*[:=]\s*([^\s#]+)`)
	reHighEntropy     = regexp.MustCompile(`\b[A-Za-z0-9/+_-]{32,}\b`)
)

const replacement = "[REDACTED]"

// Redact replaces AWS keys, PEM blocks, .env-style secret assignments, and
// high-entropy tokens. Transcripts are treated as secret-bearing: call this
// before store or print.
func Redact(s string) string {
	if s == "" {
		return s
	}
	s = rePEM.ReplaceAllString(s, "[REDACTED PEM]")
	s = rePEMOneline.ReplaceAllString(s, "[REDACTED PEM]")
	s = reAWSKey.ReplaceAllString(s, replacement)
	s = reAWSSecretAssign.ReplaceAllString(s, "aws_secret_access_key="+replacement)
	s = reAnthropic.ReplaceAllString(s, replacement)
	s = reOpenAI.ReplaceAllString(s, replacement)
	s = reGitHubFine.ReplaceAllString(s, replacement)
	s = reGitHubPAT.ReplaceAllString(s, replacement)
	s = reSlack.ReplaceAllString(s, replacement)
	s = reJWT.ReplaceAllString(s, replacement)
	s = reEnvAssign.ReplaceAllString(s, "${1}="+replacement)
	s = reGenericAssign.ReplaceAllStringFunc(s, func(m string) string {
		parts := reGenericAssign.FindStringSubmatch(m)
		if len(parts) < 2 {
			return replacement
		}
		return parts[1] + "=" + replacement
	})
	s = reHighEntropy.ReplaceAllStringFunc(s, func(tok string) string {
		if !looksSecretToken(tok) {
			return tok
		}
		return replacement
	})
	return s
}

func looksSecretToken(tok string) bool {
	if len(tok) < 32 {
		return false
	}
	if strings.Contains(tok, "/") || strings.Contains(tok, "\\") {
		return false
	}
	var classes int
	var letters, digits, upper, lower, punct int
	for _, r := range tok {
		switch {
		case unicode.IsUpper(r):
			upper++
			letters++
		case unicode.IsLower(r):
			lower++
			letters++
		case unicode.IsDigit(r):
			digits++
		case r == '/' || r == '+' || r == '_' || r == '-':
			punct++
		default:
			return false
		}
	}
	if letters > 0 {
		classes++
	}
	if digits > 0 {
		classes++
	}
	if punct > 0 {
		classes++
	}
	if upper > 0 && lower > 0 {
		classes++
	}
	if isHex(tok) && (len(tok) == 32 || len(tok) == 40 || len(tok) == 64) {
		return false
	}
	return classes >= 3 && digits >= 4
}

func isHex(s string) bool {
	for _, r := range s {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')) {
			return false
		}
	}
	return true
}
