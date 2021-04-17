package client

func PathWithKey(url, key string) string {
	return url + "/key/" + key
}

func PathWithHttp(url string) string {
	return "http://" + url
}

func PathWithPing(url string) string {
	return url + "/ping"
}
