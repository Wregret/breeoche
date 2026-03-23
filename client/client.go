package client

import (
	"bytes"
	"errors"
	"io/ioutil"
	"net/http"
)

const (
	defaultServerAddr = "localhost:15213"
	contentType       = "text/plain"
	maxRedirects      = 3
)

// Client is a simple HTTP client for Breeoche.
type Client struct {
	addr       string
	httpClient *http.Client
}

func NewClient(addr string) *Client {
	if addr == "" {
		addr = defaultServerAddr
	}
	return &Client{addr: addr, httpClient: &http.Client{}}
}

func (c *Client) Ping() (string, error) {
	resBody, status, err := c.doRequest(http.MethodGet, PathWithPing(PathWithHttp(c.addr)), nil)
	if err != nil {
		return "", err
	}
	if status != http.StatusOK {
		return "", errors.New(string(resBody))
	}
	return string(resBody), nil
}

func (c *Client) Get(key string) (string, error) {
	url := PathWithKey(PathWithHttp(c.addr), key)
	resBody, status, err := c.doRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	if status != http.StatusOK {
		return "", errors.New(string(resBody))
	}
	return string(resBody), nil
}

func (c *Client) Set(key string, value string) error {
	url := PathWithKey(PathWithHttp(c.addr), key)
	body := []byte(value)
	resBody, status, err := c.doRequest(http.MethodPost, url, body)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return errors.New(string(resBody))
	}
	return nil
}

func (c *Client) Insert(key string, value string) error {
	url := PathWithKey(PathWithHttp(c.addr), key)
	body := []byte(value)
	resBody, status, err := c.doRequest(http.MethodPut, url, body)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return errors.New(string(resBody))
	}
	return nil
}

func (c *Client) Delete(key string) error {
	url := PathWithKey(PathWithHttp(c.addr), key)
	resBody, status, err := c.doRequest(http.MethodDelete, url, nil)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return errors.New(string(resBody))
	}
	return nil
}

func (c *Client) doRequest(method, url string, body []byte) ([]byte, int, error) {
	currentURL := url
	payload := body
	for i := 0; i < maxRedirects; i++ {
		var reader *bytes.Reader
		if payload != nil {
			reader = bytes.NewReader(payload)
		} else {
			reader = bytes.NewReader(nil)
		}
		req, err := http.NewRequest(method, currentURL, reader)
		if err != nil {
			return nil, 0, err
		}
		if method == http.MethodPost || method == http.MethodPut || method == http.MethodDelete {
			req.Header.Set("Content-Type", contentType)
		}
		res, err := c.httpClient.Do(req)
		if err != nil {
			return nil, 0, err
		}
		data, err := ioutil.ReadAll(res.Body)
		res.Body.Close()
		if err != nil {
			return nil, 0, err
		}
		if res.StatusCode == http.StatusTemporaryRedirect || res.StatusCode == http.StatusPermanentRedirect {
			location := res.Header.Get("Location")
			if location == "" {
				return data, res.StatusCode, errors.New("redirect without location")
			}
			currentURL = location
			continue
		}
		return data, res.StatusCode, nil
	}
	return nil, 0, errors.New("too many redirects")
}
