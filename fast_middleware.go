// File: l/fast_middleware.go
package l

import (
	"context"
	"time"

	"github.com/valyala/fasthttp"
)

// FastLoggingMiddleware returns a fasthttp.RequestHandler middleware that logs
// incoming requests and their durations using the default logger.
// When debug-level logging is enabled, it logs additional details such as request
// headers, query parameters, cookies, and a snippet of the request body, as well as
// response headers.
func FastLoggingMiddleware(next fasthttp.RequestHandler) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		start := time.Now()
		logger := GetDefaultLogger()

		// Basic log: Incoming request
		reqMethod := string(ctx.Method())
		reqURI := string(ctx.RequestURI())
		remoteAddr := ctx.RemoteAddr().String()

		logger.Info("Incoming request",
			"method", reqMethod,
			"url", reqURI,
			"remote_addr", remoteAddr,
		)

		// If debug logging is enabled, log extra request details.
		if dbg, ok := logger.(interface {
			Enabled(context.Context, Level) bool
		}); ok && dbg.Enabled(context.Background(), LevelDebug) {
			// Collect request headers.
			headers := make(map[string]string)
			ctx.Request.Header.VisitAll(func(key, value []byte) {
				headers[string(key)] = string(value)
			})

			// Get query string.
			query := string(ctx.QueryArgs().QueryString())

			// Collect request cookies.
			cookies := make(map[string]string)
			ctx.Request.Header.VisitAllCookie(func(key, value []byte) {
				cookies[string(key)] = string(value)
			})

			// Retrieve a snippet of the request body (if any).
			var bodySnippet string
			pb := ctx.PostBody()
			if len(pb) > 0 {
				if len(pb) > 100 {
					bodySnippet = string(pb[:100]) + "..."
				} else {
					bodySnippet = string(pb)
				}
			}

			logger.Debug("Request details",
				"headers", headers,
				"query", query,
				"cookies", cookies,
				"body_snippet", bodySnippet,
			)
		}

		// Process the request.
		next(ctx)

		duration := time.Since(start)

		// Basic log: Completed request.
		logger.Info("Completed request",
			"method", reqMethod,
			"url", reqURI,
			"status", ctx.Response.StatusCode(),
			"duration", duration.String(),
		)

		// If debug logging is enabled, log extra response details.
		if dbg, ok := logger.(interface {
			Enabled(context.Context, Level) bool
		}); ok && dbg.Enabled(context.Background(), LevelDebug) {
			// Collect response headers.
			respHeaders := make(map[string]string)
			ctx.Response.Header.VisitAll(func(key, value []byte) {
				respHeaders[string(key)] = string(value)
			})

			// Optionally, log the Content-Length header.
			contentLength := string(ctx.Response.Header.Peek("Content-Length"))

			logger.Debug("Response details",
				"content_length", contentLength,
				"headers", respHeaders,
			)
		}
	}
}
