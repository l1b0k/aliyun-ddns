package myip

// https://api.myip.com/

import (
	"fmt"

	"github.com/go-resty/resty"
	"github.com/l1b0k/aliyun-ddns/addr/types"
)

type MyIP struct {
}

func (i *MyIP) Fetch() (*types.Addr, error) {
	var result Result
	client := resty.New()

	resp, err := client.R().SetResult(&result).Get("https://api.myip.com")
	if err != nil {
		return nil, err
	}
	if !resp.IsSuccess() {
		return nil, fmt.Errorf("resp error %s\n", resp.String())
	}
	return types.NewAddrFromSlice([]string{result.IP}), nil
}

type Result struct {
	IP string `json:"ip"`
}

func init() {
	types.Plugins["myip"] = func() types.Interface {
		return &MyIP{}
	}
}
