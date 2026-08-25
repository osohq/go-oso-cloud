# Changelog

## 2.5.0

- Add `ClientOptions.HttpClient` and `NewClientWithOptions()`, letting you supply a custom `*http.Client` (e.g. for custom timeouts, proxies, TLS configuration, or a custom `http.RoundTripper` for tracing). Oso's own retry/backoff behavior continues to run on top of the supplied client. If a fallback URL is also configured, the custom client is reused for fallback requests.
