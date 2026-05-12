package proxy

import (
	"net/http"
	"net/http/httputil"
	"net/url"

	"go.uber.org/zap"
)

type ReverseProxy struct {
	proxy *httputil.ReverseProxy
	log   *zap.Logger
}

func New(target string, log *zap.Logger) (*ReverseProxy, error) {
	u, err := url.Parse(target)
	if err != nil {
		return nil, err
	}

	rp := &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			pr.SetURL(u)
			pr.Out.Host = u.Host

			if id := pr.In.Header.Get("X-User-ID"); id != "" {
				pr.Out.Header.Set("X-User-ID", id)
			}
			if role := pr.In.Header.Get("X-User-Role"); role != "" {
				pr.Out.Header.Set("X-User-Role", role)
			}
			if email := pr.In.Header.Get("X-User-Email"); email != "" {
				pr.Out.Header.Set("X-User-Email", email)
			}
			if trace := pr.In.Header.Get("X-Trace-ID"); trace != "" {
				pr.Out.Header.Set("X-Trace-ID", trace)
			}
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			log.Error("Proxy error",
				zap.String("path", r.URL.Path),
				zap.Error(err),
			)
			http.Error(w, `{"error":{"code":"service_unavailable","message":"backend service unavailable"}}`, http.StatusBadGateway)
		},
	}

	return &ReverseProxy{proxy: rp, log: log}, nil
}

func (p *ReverseProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p.proxy.ServeHTTP(w, r)
}
