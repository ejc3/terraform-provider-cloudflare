package logging_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	providerlogging "github.com/cloudflare/terraform-provider-cloudflare/internal/logging"
	"github.com/hashicorp/terraform-plugin-log/tflogtest"
)

func TestRedactingMiddlewareRedactsSensitiveJSONAndHeaders(t *testing.T) {
	t.Parallel()

	const (
		requestAuthorization  = "Bearer request-authorization-secret"
		responseAuthorization = "Bearer response-authorization-secret"
		requestBody           = `{"name":"visible-request","nested":{"Value":"request-body-secret"},"items":[{"value":"array-body-secret"}]}`
		responseBody          = `{"success":true,"result":{"name":"visible-response","value":"response-body-secret"}}`
	)

	var logs bytes.Buffer
	ctx := tflogtest.RootLogger(context.Background(), &logs)
	req := httptest.NewRequest(http.MethodPost, "https://api.cloudflare.test/user/tokens", strings.NewReader(requestBody))
	req.Header.Set("Authorization", requestAuthorization)

	var receivedRequestBody string
	resp, err := providerlogging.RedactingMiddleware(ctx, "value")(req, func(req *http.Request) (*http.Response, error) {
		body, readErr := io.ReadAll(req.Body)
		if readErr != nil {
			return nil, readErr
		}
		receivedRequestBody = string(body)

		return &http.Response{
			StatusCode: http.StatusCreated,
			Status:     "201 Created",
			Proto:      "HTTP/1.1",
			Header: http.Header{
				"Authorization": []string{responseAuthorization},
				"Content-Type":  []string{"application/json"},
			},
			Body: io.NopCloser(strings.NewReader(responseBody)),
		}, nil
	})
	if err != nil {
		t.Fatalf("middleware returned an error: %v", err)
	}

	if receivedRequestBody != requestBody {
		t.Fatalf("request body changed before transport: got %q, want %q", receivedRequestBody, requestBody)
	}
	returnedResponseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("reading returned response body: %v", err)
	}
	if string(returnedResponseBody) != responseBody {
		t.Fatalf("response body changed before decoding: got %q, want %q", returnedResponseBody, responseBody)
	}

	logOutput := logs.String()
	for _, secret := range []string{
		requestAuthorization,
		responseAuthorization,
		"request-body-secret",
		"array-body-secret",
		"response-body-secret",
	} {
		if strings.Contains(logOutput, secret) {
			t.Errorf("log output contains secret %q: %s", secret, logOutput)
		}
	}
	for _, visible := range []string{"visible-request", "visible-response", "[redacted]"} {
		if !strings.Contains(logOutput, visible) {
			t.Errorf("log output does not retain %q: %s", visible, logOutput)
		}
	}
}

func TestRedactingMiddlewareOmitsMalformedJSONBodies(t *testing.T) {
	t.Parallel()

	const (
		requestBody  = `{"name":"request-before-secret","value":"malformed-request-secret"`
		responseBody = `{"name":"response-before-secret","value":"malformed-response-secret" trailing`
	)

	var logs bytes.Buffer
	ctx := tflogtest.RootLogger(context.Background(), &logs)
	req := httptest.NewRequest(http.MethodPost, "https://api.cloudflare.test/user/tokens", strings.NewReader(requestBody))

	var receivedRequestBody string
	resp, err := providerlogging.RedactingMiddleware(ctx, "value")(req, func(req *http.Request) (*http.Response, error) {
		body, readErr := io.ReadAll(req.Body)
		if readErr != nil {
			return nil, readErr
		}
		receivedRequestBody = string(body)

		return &http.Response{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
			Proto:      "HTTP/1.1",
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(responseBody)),
		}, nil
	})
	if err != nil {
		t.Fatalf("middleware returned an error: %v", err)
	}

	if receivedRequestBody != requestBody {
		t.Fatalf("malformed request body changed before transport: got %q, want %q", receivedRequestBody, requestBody)
	}
	returnedResponseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("reading returned response body: %v", err)
	}
	if string(returnedResponseBody) != responseBody {
		t.Fatalf("malformed response body changed before decoding: got %q, want %q", returnedResponseBody, responseBody)
	}

	logOutput := logs.String()
	for _, bodyContent := range []string{
		"malformed-request-secret",
		"malformed-response-secret",
		"request-before-secret",
		"response-before-secret",
	} {
		if strings.Contains(logOutput, bodyContent) {
			t.Errorf("log output contains content from malformed JSON body %q: %s", bodyContent, logOutput)
		}
	}
	if count := strings.Count(logOutput, "[body omitted: invalid JSON]"); count != 2 {
		t.Errorf("log output contains %d malformed-body markers, want 2: %s", count, logOutput)
	}
}
