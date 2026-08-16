package logging_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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
	req := httptest.NewRequest(http.MethodPost, "https://api.cloudflare.test/user/tokens", nil)
	requestBodyReader := &trackingReadCloser{Reader: strings.NewReader(requestBody)}
	req.Body = requestBodyReader
	req.Header.Set("Authorization", requestAuthorization)

	var receivedRequestBody string
	var responseBodyReader *trackingReadCloser
	resp, err := providerlogging.RedactingMiddleware(ctx, "value")(req, func(req *http.Request) (*http.Response, error) {
		if !requestBodyReader.closed {
			t.Error("original request body was not closed before the transport ran")
		}
		body, readErr := io.ReadAll(req.Body)
		if readErr != nil {
			return nil, readErr
		}
		receivedRequestBody = string(body)

		responseBodyReader = &trackingReadCloser{Reader: strings.NewReader(responseBody)}
		return &http.Response{
			StatusCode: http.StatusCreated,
			Status:     "201 Created",
			Proto:      "HTTP/1.1",
			Header: http.Header{
				"Authorization": []string{responseAuthorization},
				"Content-Type":  []string{"application/json"},
			},
			Body: responseBodyReader,
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
	if !requestBodyReader.closed || responseBodyReader == nil || !responseBodyReader.closed {
		t.Fatalf("original bodies were not closed: request=%t response=%v", requestBodyReader.closed, responseBodyReader)
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

func TestRedactingMiddlewareRedactsSensitiveHeadersCaseInsensitively(t *testing.T) {
	t.Parallel()

	const (
		requestCookie             = "session=request-cookie-secret"
		requestProxyAuthorization = "Basic request-proxy-authorization-secret"
		responseCookie            = "session=response-cookie-secret; Secure; HttpOnly"
	)

	var logs bytes.Buffer
	ctx := tflogtest.RootLogger(context.Background(), &logs)
	req := httptest.NewRequest(http.MethodGet, "https://api.cloudflare.test/test", nil)
	req.Header["cOoKiE"] = []string{requestCookie}
	req.Header["pRoXy-AuThOrIzAtIoN"] = []string{requestProxyAuthorization}

	_, err := providerlogging.RedactingMiddleware(ctx)(req, func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
			Proto:      "HTTP/1.1",
			Header: http.Header{
				"sEt-CoOkIe": []string{responseCookie},
			},
		}, nil
	})
	if err != nil {
		t.Fatalf("middleware returned an error: %v", err)
	}

	logOutput := logs.String()
	for _, secret := range []string{requestCookie, requestProxyAuthorization, responseCookie} {
		if strings.Contains(logOutput, secret) {
			t.Errorf("log output contains secret %q: %s", secret, logOutput)
		}
	}
	var decodedMessages strings.Builder
	decoder := json.NewDecoder(strings.NewReader(logOutput))
	for {
		var entry struct {
			Message string `json:"@message"`
		}
		if err := decoder.Decode(&entry); errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			t.Fatalf("decode log entry: %v", err)
		}
		decodedMessages.WriteString(entry.Message)
	}
	for _, header := range []string{
		"> cookie: [redacted]",
		"> proxy-authorization: [redacted]",
		"< set-cookie: [redacted]",
	} {
		if !strings.Contains(decodedMessages.String(), header) {
			t.Errorf("log output does not contain redacted header %q: %s", header, logOutput)
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

func TestLoggingClosesBodyWhenReadFails(t *testing.T) {
	t.Parallel()

	requestBody := &failingReadCloser{}
	req := httptest.NewRequest(http.MethodPost, "https://api.cloudflare.test/test", nil)
	req.Body = requestBody
	if err := providerlogging.LogRequest(context.Background(), req); err == nil {
		t.Fatal("request read failure must be returned")
	}
	if !requestBody.closed {
		t.Fatal("request body was not closed after a read failure")
	}

	responseBody := &failingReadCloser{}
	resp := &http.Response{Body: responseBody, Header: make(http.Header)}
	if err := providerlogging.LogResponse(context.Background(), resp); err == nil {
		t.Fatal("response read failure must be returned")
	}
	if !responseBody.closed {
		t.Fatal("response body was not closed after a read failure")
	}
}

type trackingReadCloser struct {
	io.Reader
	closed bool
}

func (body *trackingReadCloser) Close() error {
	body.closed = true
	return nil
}

type failingReadCloser struct{ closed bool }

func (*failingReadCloser) Read([]byte) (int, error) { return 0, errors.New("read failed") }

func (body *failingReadCloser) Close() error {
	body.closed = true
	return nil
}
