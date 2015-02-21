package goparse

import (
	"net/url"
	"os"
	"testing"
)

type Blog struct {
	objectId string `json:"objectId"`
	Date     string `json:"Date"`
	HashId   string `json:"HashId"`
	Name     string `json:"Name"`
	Title    string `json:"Title"`
	Url      string `json:"Url"`
	UserId   int    `json:"UserId"`
}

var (
	APPID  = os.Getenv("PARSE_APP_ID")
	APPKEY = os.Getenv("PARSE_APP_KEY")
)

func TestRetrieve(t *testing.T) {
	cli := ParseClient(APPID, APPKEY, nil)
	v := url.Values{}
	var blog Blog
	res, err := cli.Retrieve("blog", "wq63xtmGJL", v, &blog)
	if err != nil {
		t.Errorf(err.Error())
	}
	b, _ := res.(*Blog)
	t.Logf("success: Name is %s", b.Name)
}

func TestRetrieveObjectNotFound(t *testing.T) {
	cli := ParseClient(APPID, APPKEY, nil)
	v := url.Values{}
	var blog Blog
	res, err := cli.Retrieve("blog", "testKey", v, &blog)
	if err != nil {
		t.Logf(err.Error())
	} else {
		t.Errorf("Failed auth error: %s", res)
	}
}

func TestAuthError(t *testing.T) {
	cli := ParseClient(APPID, "testKey", nil)
	v := url.Values{}
	var blog Blog
	res, err := cli.Retrieve("blog", "wq63xtmGJL", v, &blog)
	if err != nil {
		t.Logf(err.Error())
	} else {
		t.Errorf("Failed auth error: %s", res)
	}
}
