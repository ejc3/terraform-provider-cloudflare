package logging

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"

	"github.com/cloudflare/cloudflare-go/v7/option"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const redactedValue = "[redacted]"

var sensitiveHeaderNames = []string{"x-auth-email", "x-auth-key", "x-auth-user-service-key", "authorization"}

func Middleware(ctx context.Context) option.Middleware {
	return middleware(ctx, nil)
}

// RedactingMiddleware logs requests and responses while recursively replacing
// values for the configured JSON field names. If a body is not valid JSON, the
// entire body is omitted from the log because it cannot be safely inspected for
// sensitive fields. The original request and response bodies are restored after
// logging so the HTTP transport and response decoder can still consume them.
func RedactingMiddleware(ctx context.Context, sensitiveJSONFields ...string) option.Middleware {
	fields := make(map[string]struct{}, len(sensitiveJSONFields))
	for _, field := range sensitiveJSONFields {
		if field = strings.TrimSpace(field); field != "" {
			fields[strings.ToLower(field)] = struct{}{}
		}
	}

	return middleware(ctx, fields)
}

func middleware(ctx context.Context, sensitiveJSONFields map[string]struct{}) option.Middleware {
	return func(req *http.Request, next option.MiddlewareNext) (*http.Response, error) {
		if req != nil {
			if err := logRequest(ctx, req, sensitiveJSONFields); err != nil {
				return nil, err
			}
		}

		resp, err := next(req)

		if resp != nil {
			if err := logResponse(ctx, resp, sensitiveJSONFields); err != nil {
				return nil, err
			}
		}

		return resp, err
	}
}

func LogRequest(ctx context.Context, req *http.Request) error {
	return logRequest(ctx, req, nil)
}

func logRequest(ctx context.Context, req *http.Request, sensitiveJSONFields map[string]struct{}) error {
	lines := []string{fmt.Sprintf("\n%s %s %s", req.Method, req.URL.Path, req.Proto)}

	// Log headers
	for name, values := range req.Header {
		for _, value := range values {

			if slices.Contains(sensitiveHeaderNames, strings.ToLower(name)) {
				value = redactedValue
			}

			lines = append(lines, fmt.Sprintf("> %s: %s", strings.ToLower(name), value))
		}
	}

	if req.Body != nil {
		// Read the body without mutating the original response
		bodyBytes, err := io.ReadAll(req.Body)
		if err != nil {
			return err
		}

		// Restore the original body to the response so it can be read again
		req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

		// Log a sanitized copy of the body. The original bytes remain on req.
		lines = append(lines, ">\n", string(redactJSONBody(bodyBytes, sensitiveJSONFields)), "\n")
	}

	tflog.Debug(ctx, strings.Join(lines, "\n"))

	return nil
}

func LogResponse(ctx context.Context, resp *http.Response) error {
	return logResponse(ctx, resp, nil)
}

func logResponse(ctx context.Context, resp *http.Response, sensitiveJSONFields map[string]struct{}) error {
	// Log the status code
	lines := []string{fmt.Sprintf("\n< %s %s", resp.Proto, resp.Status)}

	// Log headers
	for name, values := range resp.Header {
		for _, value := range values {
			if slices.Contains(sensitiveHeaderNames, strings.ToLower(name)) {
				value = redactedValue
			}

			lines = append(lines, fmt.Sprintf("< %s: %s", strings.ToLower(name), value))
		}
	}

	if resp.Body != nil {
		// Read the body without mutating the original response
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		// Restore the original body to the response so it can be read again
		resp.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

		lines = append(lines, "<\n", string(redactJSONBody(bodyBytes, sensitiveJSONFields)), "\n")
	}

	// Log the body
	tflog.Debug(ctx, strings.Join(lines, "\n"))

	return nil
}

func redactJSONBody(body []byte, sensitiveJSONFields map[string]struct{}) []byte {
	if len(body) == 0 || len(sensitiveJSONFields) == 0 {
		return body
	}

	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()

	var decoded any
	if err := decoder.Decode(&decoded); err != nil {
		return []byte("[body omitted: invalid JSON]")
	}

	// Reject trailing non-whitespace or a second JSON value. Logging a partial
	// parse could expose a secret in the unparsed suffix.
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return []byte("[body omitted: invalid JSON]")
	}

	redactJSONValue(decoded, sensitiveJSONFields)
	redacted, err := json.Marshal(decoded)
	if err != nil {
		return []byte("[body omitted: invalid JSON]")
	}

	return redacted
}

func redactJSONValue(value any, sensitiveJSONFields map[string]struct{}) {
	switch value := value.(type) {
	case map[string]any:
		for key, nested := range value {
			if _, sensitive := sensitiveJSONFields[strings.ToLower(key)]; sensitive {
				value[key] = redactedValue
				continue
			}

			redactJSONValue(nested, sensitiveJSONFields)
		}
	case []any:
		for _, nested := range value {
			redactJSONValue(nested, sensitiveJSONFields)
		}
	}
}
