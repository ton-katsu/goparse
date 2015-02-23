package goparse

import (
	"bytes"
	"encoding/json"
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

type BlogResults struct {
	Results []Blog `json:"results"`
}

var (
	APPID  = os.Getenv("PARSE_APP_ID")
	APPKEY = os.Getenv("PARSE_APP_KEY")
)

func TestRetrieveObjectNotFound(t *testing.T) {
	cli := Client(APPID, APPKEY, nil)
	v := url.Values{}
	var blog Blog
	res, err := cli.RetrieveObject("devBlog", "testKey", v, &blog)
	if err != nil {
		t.Logf(err.Error())
	} else {
		t.Errorf("Failed auth error: %s", res)
	}
}

func TestAuthError(t *testing.T) {
	cli := Client(APPID, "testKey", nil)
	v := url.Values{}
	var blog Blog
	res, err := cli.RetrieveObject("devBlog", "wq63xtmGJL", v, &blog)
	if err != nil {
		t.Logf(err.Error())
	} else {
		t.Errorf("Failed auth error: %s", res)
	}
}

func TestRetrieveObject(t *testing.T) {
	cli := Client(APPID, APPKEY, nil)
	v := url.Values{}
	var b Blog
	res, err := cli.RetrieveObject("devBlog", "bPVTnpZOEL", v, &b)
	if err != nil {
		t.Errorf(err.Error())
	} else {
		blog, _ := res.(*Blog)
		t.Logf("success: Name is %s", blog.Name)
	}
}

func TestRetrieveObjects(t *testing.T) {
	cli := Client(APPID, APPKEY, nil)
	v := url.Values{}
	v.Set("limit", "2")
	con, _ := json.Marshal(map[string]string{"Name": "佐藤　麗奈"})
	v.Add("where", string(con))
	var b BlogResults
	res, err := cli.RetrieveObjects("devBlog", v, &b)
	if err != nil {
		t.Errorf(err.Error())
	} else {
		blogs, _ := res.(*BlogResults)
		for _, blog := range blogs.Results {
			t.Logf("success: Name is %s", blog.Name)
		}
	}

}

var ObjectId string

func TestCreateObject(t *testing.T) {
	cli := Client(APPID, APPKEY, nil)
	blog := Blog{
		Name:   "佐藤　麗奈",
		HashId: "aabbcc",
		Title:  "create test",
		Url:    "https://example.com",
		UserId: 22,
	}
	con, _ := json.Marshal(&blog)
	res, err := cli.CreateObject("devBlog", bytes.NewReader(con))
	if err != nil {
		t.Errorf(err.Error())
	} else {
		t.Logf("success, objectId: %s, createdAt: %s", res.ObjectId, res.CreatedAt)
		ObjectId = res.ObjectId
	}
}

func TestUpdateObject(t *testing.T) {
	cli := Client(APPID, APPKEY, nil)
	blog := Blog{
		Name:   "佐藤　麗奈",
		HashId: "aabbcc",
		Title:  "update test",
		Url:    "https://example.com",
		UserId: 22,
	}
	con, _ := json.Marshal(&blog)
	res, err := cli.UpdateObject("devBlog", ObjectId, bytes.NewReader(con))
	if err != nil {
		t.Errorf(err.Error())
	} else {
		t.Logf("success, objectId: %s, updateAt: %s", ObjectId, res.UpdatedAt)
	}
}

func TestDeleteObject(t *testing.T) {
	cli := Client(APPID, APPKEY, nil)
	err := cli.DeleteObject("devBlog", ObjectId)
	if err != nil {
		t.Errorf(err.Error())
	} else {
		t.Log("success")
	}
}
