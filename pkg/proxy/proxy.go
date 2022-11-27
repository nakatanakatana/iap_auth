package proxy

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"

	"github.com/gojekfarm/iap_auth/httputil"
	"github.com/gojekfarm/iap_auth/pkg/logger"
)

type Proxy interface {
	http.Handler
	Address() string
}

func singleJoiningSlash(a, b string) string {
	aslash := strings.HasSuffix(a, "/")
	bslash := strings.HasPrefix(b, "/")
	switch {
	case aslash && bslash:
		return a + b[1:]
	case !aslash && !bslash:
		return a + "/" + b
	}
	return a + b
}

func joinURLPath(a, b *url.URL) (path, rawpath string) {
	if a.RawPath == "" && b.RawPath == "" {
		return singleJoiningSlash(a.Path, b.Path), ""
	}
	// Same as singleJoiningSlash, but uses EscapedPath to determine
	// whether a slash should be added
	apath := a.EscapedPath()
	bpath := b.EscapedPath()

	aslash := strings.HasSuffix(apath, "/")
	bslash := strings.HasPrefix(bpath, "/")

	switch {
	case aslash && bslash:
		return a.Path + b.Path[1:], apath + bpath[1:]
	case !aslash && !bslash:
		return a.Path + "/" + b.Path, apath + "/" + bpath
	}
	return a.Path + b.Path, apath + bpath
}

func CreateRewrite(target *url.URL, atomictoken *atomic.Value) func(*httputil.ProxyRequest) {
	return func(r *httputil.ProxyRequest) {
		// rewriteRequestURL
		targetQuery := target.RawQuery
		r.Out.URL.Scheme = target.Scheme
		r.Out.URL.Host = target.Host
		r.Out.URL.Path, r.Out.URL.RawPath = joinURLPath(target, r.Out.URL)
		if targetQuery == "" || r.Out.URL.RawQuery == "" {
			r.Out.URL.RawQuery = targetQuery + r.Out.URL.RawQuery
		} else {
			r.Out.URL.RawQuery = targetQuery + "&" + r.Out.URL.RawQuery
		}

		if token := atomictoken.Load().(string); token != "" {
			r.Out.Header.Set("Proxy-Authorization", fmt.Sprintf("Bearer %s", token))
		}
	}
}

func New(backend string, atomictoken *atomic.Value) (Proxy, error) {
	var transport http.RoundTripper
	target, err := url.Parse(backend)
	if err != nil {
		return nil, err
	}

	transport = http.DefaultTransport
	rewrite := CreateRewrite(target, atomictoken)

	return &proxy{
		Backend:     newProxyBackend(target, transport, rewrite),
		AtomicToken: atomictoken,
	}, nil
}

type proxy struct {
	Backend     *ProxyBackend
	AtomicToken *atomic.Value
}

func (prx *proxy) Address() string {
	return prx.Backend.URL().String()
}

func (prx *proxy) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	log.Printf("h=%+v", req.Header)
	token := prx.AtomicToken.Load().(string)
	logger.Debugf("injecting token %s", fmt.Sprintf("Bearer %s", token))
	req.Host = prx.Backend.URL().Host
	req.URL.Scheme = prx.Backend.URL().Scheme
	logger.Debugf("Request URL  %s", req.URL)
	req.Header.Set("Host", prx.Address())
	req.Header.Set("Proxy-Authorization", fmt.Sprintf("Bearer %s", token))

	log.Printf("h=%+v", req.Header)
	prx.Backend.ServeHTTP(rw, req)
}
