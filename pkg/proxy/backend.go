package proxy

import (
	"net/http"
	"net/url"

	"github.com/gojekfarm/iap_auth/httputil"
)

type ProxyBackend struct {
	url   *url.URL
	proxy *httputil.ReverseProxy
}

func (this *ProxyBackend) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	this.proxy.ServeHTTP(rw, req)
}

func (this *ProxyBackend) URL() *url.URL {
	return this.url
}

func newProxyBackend(backendURL *url.URL, transport http.RoundTripper, rewrite func(*httputil.ProxyRequest)) *ProxyBackend {
	// proxy := httputil.NewSingleHostReverseProxy(backendURL)
	proxy := &httputil.ReverseProxy{
		Rewrite: rewrite,
	}
	proxy.Transport = transport

	return &ProxyBackend{
		url:   backendURL,
		proxy: proxy,
	}
}
