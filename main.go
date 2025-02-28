package main

import (
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/pkg/errors"
)

type Client struct{ url string }

func NewClient(url string) Client { return Client{url} }
func (c Client) MakeRequest() (string, error) {
	res, err := http.Get(c.url + "/users")
	if err != nil {
		return "", errors.Wrap(err, "An error occured while making the request")
	}

	defer res.Body.Close()
	out, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", errors.Wrap(err, "An error occured when reading the response")
	}
	return string(out), nil
}

func main() {
	client := NewClient("https://gorest.co.in/public/v2/")
	resp, err := client.MakeRequest()
	if err != nil {
		panic(err)
	}
	fmt.Println(resp)
}
