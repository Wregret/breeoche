package client

import (
	"errors"
	"io/ioutil"
	"net/http"
	"strings"
)

const (
	serverAddr  = "localhost:15213"
	contentType = "text/plain"
)

type client struct {
	httpClient *http.Client
}

func NewClient() *client {
	c := &client{httpClient: &http.Client{}}
	return c
}

func (c *client) Ping() (string, error) {
	url := PathWithHttp(serverAddr)
	url = PathWithPing(url)
	res, err := c.httpClient.Get(url)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	value, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	return string(value), nil
}

func (c *client) Get(key string) (string, error) {
	url := PathWithHttp(serverAddr)
	url = PathWithKey(url, key)
	res, err := c.httpClient.Get(url)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	msg, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	if res.StatusCode != http.StatusOK {
		return "", errors.New(string(msg))
	}
	return string(msg), nil
}

func (c *client) Set(key string, value string) error {
	url := PathWithHttp(serverAddr)
	url = PathWithKey(url, key)
	res, err := c.httpClient.Post(url, contentType, strings.NewReader(value))
	defer res.Body.Close()
	if err != nil {
		return err
	}
	defer res.Body.Close()
	msg, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}
	if res.StatusCode != http.StatusOK {
		return errors.New(string(msg))
	}
	return nil
}

func (c *client) Insert(key string, value string) error {
	url := PathWithHttp(serverAddr)
	url = PathWithKey(url, key)
	req, err := http.NewRequest(http.MethodPut, url, strings.NewReader(value))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", contentType)
	res, err := c.httpClient.Do(req)
	defer res.Body.Close()
	if err != nil {
		return err
	}
	defer res.Body.Close()
	msg, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}
	if res.StatusCode != http.StatusOK {
		return errors.New(string(msg))
	}
	return nil
}

func (c *client) Delete(key string) error {
	url := PathWithHttp(serverAddr)
	url = PathWithKey(url, key)
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", contentType)
	res, err := c.httpClient.Do(req)
	defer res.Body.Close()
	if err != nil {
		return err
	}
	defer res.Body.Close()
	msg, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}
	if res.StatusCode != http.StatusOK {
		return errors.New(string(msg))
	}
	return nil
}
