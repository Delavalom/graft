package bedrock

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"
)

const (
	sigV4Algorithm = "AWS4-HMAC-SHA256"
	sigV4Service   = "bedrock"
	sigV4Request   = "aws4_request"
	amzDateFormat  = "20060102T150405Z"
	dateFormat     = "20060102"
)

type credentials struct {
	accessKey    string
	secretKey    string
	sessionToken string
}

// signRequest signs an HTTP request with AWS Signature Version 4.
func signRequest(req *http.Request, body []byte, creds credentials, region string) {
	// Use existing X-Amz-Date if set (for test determinism), otherwise generate one.
	amzDate := req.Header.Get("X-Amz-Date")
	if amzDate == "" {
		amzDate = time.Now().UTC().Format(amzDateFormat)
		req.Header.Set("X-Amz-Date", amzDate)
	}

	dateStamp := amzDate[:8]

	// Set security token header if present.
	if creds.sessionToken != "" {
		req.Header.Set("X-Amz-Security-Token", creds.sessionToken)
	}

	// Determine host value.
	host := req.Host
	if host == "" {
		host = req.URL.Host
	}

	// Build signed headers list.
	signedHeaderKeys := []string{"content-type", "host", "x-amz-date"}
	if creds.sessionToken != "" {
		signedHeaderKeys = append(signedHeaderKeys, "x-amz-security-token")
	}
	sort.Strings(signedHeaderKeys)
	signedHeaders := strings.Join(signedHeaderKeys, ";")

	// Build canonical headers.
	headerValues := map[string]string{
		"content-type":         req.Header.Get("Content-Type"),
		"host":                 host,
		"x-amz-date":          amzDate,
		"x-amz-security-token": creds.sessionToken,
	}
	var canonicalHeaders strings.Builder
	for _, key := range signedHeaderKeys {
		canonicalHeaders.WriteString(key)
		canonicalHeaders.WriteString(":")
		canonicalHeaders.WriteString(headerValues[key])
		canonicalHeaders.WriteString("\n")
	}

	// Build canonical query string.
	canonicalQueryString := req.URL.RawQuery

	// Build canonical request.
	payloadHash := sha256Hex(body)
	canonicalRequest := strings.Join([]string{
		req.Method,
		req.URL.Path,
		canonicalQueryString,
		canonicalHeaders.String(),
		signedHeaders,
		payloadHash,
	}, "\n")

	// Build credential scope and string to sign.
	credentialScope := fmt.Sprintf("%s/%s/%s/%s", dateStamp, region, sigV4Service, sigV4Request)
	stringToSign := strings.Join([]string{
		sigV4Algorithm,
		amzDate,
		credentialScope,
		sha256Hex([]byte(canonicalRequest)),
	}, "\n")

	// Derive signing key and compute signature.
	signingKey := deriveSigningKey(creds.secretKey, dateStamp, region)
	signature := hex.EncodeToString(hmacSHA256(signingKey, []byte(stringToSign)))

	// Set Authorization header.
	authHeader := fmt.Sprintf("%s Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		sigV4Algorithm, creds.accessKey, credentialScope, signedHeaders, signature)
	req.Header.Set("Authorization", authHeader)
}

// deriveSigningKey derives the SigV4 signing key via HMAC chain.
func deriveSigningKey(secretKey, dateStamp, region string) []byte {
	kDate := hmacSHA256([]byte("AWS4"+secretKey), []byte(dateStamp))
	kRegion := hmacSHA256(kDate, []byte(region))
	kService := hmacSHA256(kRegion, []byte(sigV4Service))
	kSigning := hmacSHA256(kService, []byte(sigV4Request))
	return kSigning
}

// hmacSHA256 computes HMAC-SHA256.
func hmacSHA256(key, data []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}

// sha256Hex computes the SHA-256 hex digest.
func sha256Hex(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}
