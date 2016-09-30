package client

import (
	"bytes"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"time"
)

type Client interface {
	Do(*http.Request) (*http.Response, error)
}

type ClientFunc func(*http.Request) (*http.Response, error)

func (f ClientFunc) Do(r *http.Request) (*http.Response, error) {
	return f(r)
}

type Decorator func(Client) Client

func Logging(l *log.Logger) Decorator {
	return func(c Client) Client {
		return ClientFunc(func(r *http.Request) (*http.Response, error) {
			l.Printf("%s: %s %s", r.UserAgent(), r.Method, r.URL)
			return c.Do(r)
		})
	}
}

type Director func(*http.Request) error

func Proxy(d Director) Decorator {
	return func(c Client) Client {
		return ClientFunc(func(r *http.Request) (*http.Response, error) {
			if err := d(r); err != nil {
				return nil, err
			}
			return c.Do(r)
		})
	}
}

func Match(ifFunc func([]byte) bool, ifURL, elseURL string) Director {
	return func(r *http.Request) error {
		buf, err := ioutil.ReadAll(r.Body)
		if err != nil {
			return err
		}
		unchanged := ioutil.NopCloser(bytes.NewBuffer(buf))
		var decision string
		if ifFunc != nil && ifFunc(buf) {
			decision = ifURL
		} else {
			decision = elseURL
		}
		proxyURL, err := url.Parse(decision)
		if err != nil {
			return err
		}
		r.URL = proxyURL
		r.Body = unchanged
		return nil
	}
}

func FaultTolerance(attempts int, backoff time.Duration) Decorator {
	return func(c Client) Client {
		return ClientFunc(func(r *http.Request) (res *http.Response, err error) {
			for i := 0; i < attempts; i++ {
				if res, err = c.Do(r); err == nil {
					break
				}
				time.Sleep(backoff * time.Duration(i))
			}
			return res, err
		})
	}
}

func Decorate(c Client, ds ...Decorator) Client {
	decorated := c
	for _, decorate := range ds {
		decorated = decorate(decorated)
	}
	return decorated
}
