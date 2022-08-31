package clients

import (
	"encoding/base64"
	"net/url"
)

func (c *Client) SetBasicAuthFromUserInfo(userInfo *url.Userinfo) {
	if userInfo == nil {
		return
	}
	c.Headers.Add("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(userInfo.String())))
}
