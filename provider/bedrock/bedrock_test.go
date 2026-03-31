package bedrock

import (
	"net/http"
	"strings"
	"testing"
)

func TestSignRequest_SetsRequiredHeaders(t *testing.T) {
	req, err := http.NewRequest("POST", "https://bedrock-runtime.us-east-1.amazonaws.com/model/anthropic.claude-v2/invoke", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")

	creds := credentials{
		accessKey: "AKIAIOSFODNN7EXAMPLE",
		secretKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
	}

	signRequest(req, []byte(`{}`), creds, "us-east-1")

	auth := req.Header.Get("Authorization")
	if auth == "" {
		t.Fatal("Authorization header not set")
	}
	if !strings.HasPrefix(auth, "AWS4-HMAC-SHA256") {
		t.Errorf("Authorization header should have AWS4-HMAC-SHA256 prefix, got: %s", auth)
	}
	if !strings.Contains(auth, creds.accessKey) {
		t.Errorf("Authorization header should contain access key, got: %s", auth)
	}
	if !strings.Contains(auth, "us-east-1/bedrock/aws4_request") {
		t.Errorf("Authorization header should contain region/service scope, got: %s", auth)
	}

	amzDate := req.Header.Get("X-Amz-Date")
	if amzDate == "" {
		t.Fatal("X-Amz-Date header not set")
	}
	if len(amzDate) != 16 {
		t.Errorf("X-Amz-Date should be 16 chars, got %d: %s", len(amzDate), amzDate)
	}
	if amzDate[8] != 'T' {
		t.Errorf("X-Amz-Date should have T at position 8, got: %s", amzDate)
	}
	if amzDate[15] != 'Z' {
		t.Errorf("X-Amz-Date should have Z at position 15, got: %s", amzDate)
	}
}

func TestSignRequest_WithSessionToken(t *testing.T) {
	req, err := http.NewRequest("POST", "https://bedrock-runtime.us-east-1.amazonaws.com/model/anthropic.claude-v2/invoke", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")

	creds := credentials{
		accessKey:    "AKIAIOSFODNN7EXAMPLE",
		secretKey:    "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
		sessionToken: "FwoGZXIvYXdzEBYaDHqa0AP",
	}

	signRequest(req, []byte(`{}`), creds, "us-east-1")

	token := req.Header.Get("X-Amz-Security-Token")
	if token != creds.sessionToken {
		t.Errorf("X-Amz-Security-Token should be %q, got %q", creds.sessionToken, token)
	}

	auth := req.Header.Get("Authorization")
	if !strings.Contains(auth, "x-amz-security-token") {
		t.Errorf("signed headers should include x-amz-security-token, got: %s", auth)
	}
}

func TestSignRequest_DeterministicSignature(t *testing.T) {
	body := []byte(`{"prompt":"hello"}`)
	creds := credentials{
		accessKey: "AKIAIOSFODNN7EXAMPLE",
		secretKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
	}
	fixedDate := "20240101T120000Z"

	makeReq := func() *http.Request {
		req, err := http.NewRequest("POST", "https://bedrock-runtime.us-east-1.amazonaws.com/model/anthropic.claude-v2/invoke", nil)
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Amz-Date", fixedDate)
		return req
	}

	req1 := makeReq()
	signRequest(req1, body, creds, "us-east-1")

	req2 := makeReq()
	signRequest(req2, body, creds, "us-east-1")

	auth1 := req1.Header.Get("Authorization")
	auth2 := req2.Header.Get("Authorization")

	if auth1 != auth2 {
		t.Errorf("signatures should be deterministic\nfirst:  %s\nsecond: %s", auth1, auth2)
	}
}
