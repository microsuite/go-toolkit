package utils

import "net/url"

func IsValidURL(input string) bool {
	u, err := url.ParseRequestURI(input)
	if err != nil {
		return false
	}

	// 必须包含协议和主机名
	if u.Scheme == "" || u.Host == "" {
		return false
	}

	// 检查常见协议
	schemes := []string{"http", "https", "ftp", "sftp"}
	for _, s := range schemes {
		if u.Scheme == s {
			return true
		}
	}
	return false
}
