package goparse

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"

	"appengine"
	"appengine/urlfetch"
)

const (
	parseUrl          = "https://api.parse.com/1/classes/"
	parseAppIdHeader  = "X-Parse-Application-Id"
	parseApiKeyHeader = "X-Parse-REST-API-Key"
	contentTypeHeader = "Content-Type"
	contentType       = "application/json"
	_POST             = iota
	_GET
	_PUT
	_DELETE
)

type ParseApi struct {
	ParseAppId  string
	ParseApiKey string
	Client      *http.Client
	queryQueue  chan query
}

type query struct {
	method      int
	url         string
	form        url.Values
	reqData     io.Reader
	resData     interface{}
	response_ch chan response
}

type response struct {
	resData interface{}
	err     error
}

func ParseClient(AppId string, AppKey string, c appengine.Context) *ParseApi {
	queue := make(chan query)
	client := &ParseApi{}
	if c != nil {
		client = &ParseApi{
			ParseAppId:  AppId,
			ParseApiKey: AppKey,
			Client:      urlfetch.Client(c),
			queryQueue:  queue,
		}
	} else {
		client = &ParseApi{
			ParseAppId:  AppId,
			ParseApiKey: AppKey,
			Client:      http.DefaultClient,
			queryQueue:  queue,
		}
	}
	go client.throttledQuery()
	return client
}

func (c ParseApi) Create(className string, v url.Values, reqData io.Reader, resData interface{}) (interface{}, error) {
	response_ch := make(chan response)
	c.queryQueue <- query{_POST, parseUrl + className, v, reqData, &resData, response_ch}
	return resData, (<-response_ch).err
}

func (c ParseApi) Retrieve(className string, objectId string, v url.Values, resData interface{}) (interface{}, error) {
	response_ch := make(chan response)
	c.queryQueue <- query{_GET, parseUrl + className + "/" + objectId, v, nil, resData, response_ch}
	return resData, (<-response_ch).err
}

func (c ParseApi) Update(className string, objectId string, v url.Values, resData interface{}) (interface{}, error) {
	response_ch := make(chan response)
	c.queryQueue <- query{_PUT, parseUrl + className + "/" + objectId, v, nil, &resData, response_ch}
	return resData, (<-response_ch).err
}

func (c ParseApi) Delete(className string, objectId string, v url.Values, resData interface{}) (interface{}, error) {
	response_ch := make(chan response)
	c.queryQueue <- query{_DELETE, parseUrl + className + "/" + objectId, v, nil, &resData, response_ch}
	return resData, (<-response_ch).err
}

func (c ParseApi) execQuery(method int, urlStr string, form url.Values, reqData io.Reader, resData interface{}) error {
	switch method {
	case _POST:
		return c.execHttpRequest("POST", urlStr, form, reqData, resData)
	case _GET:
		return c.execHttpRequest("GET", urlStr, form, reqData, resData)
	case _PUT:
		return c.execHttpRequest("PUT", urlStr, form, reqData, resData)
	case _DELETE:
		return c.execHttpRequest("DELETE", urlStr, form, reqData, resData)
	default:
		return fmt.Errorf("HTTP method not yet supported")
	}
}

func (c *ParseApi) throttledQuery() {
	for q := range c.queryQueue {
		method := q.method
		url := q.url
		form := q.form
		reqData := q.reqData
		resData := q.resData

		response_ch := q.response_ch
		err := c.execQuery(method, url, form, reqData, resData)
		response_ch <- response{resData, err}
	}
}

func (c ParseApi) execHttpRequest(method string, urlStr string, form url.Values, reqData io.Reader, resData interface{}) error {
	query := form.Encode()
	req, _ := http.NewRequest(method, urlStr+"?"+query, reqData)
	req.Header.Set(parseAppIdHeader, c.ParseAppId)
	req.Header.Add(parseApiKeyHeader, c.ParseApiKey)
	req.Header.Add(contentTypeHeader, contentType)
	res, err := c.Client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	return decodeResponse(res, resData)
}

func decodeResponse(res *http.Response, resData interface{}) error {
	if (res.StatusCode != 200) && (res.StatusCode != 201) {
		return newApiError(res)
	}
	return json.NewDecoder(res.Body).Decode(resData)
}

type ParseErrorResponse struct {
	Code  int    `json:"code"`
	Error string `json:"error"`
}

// ApiError is an error implementation
type ApiError struct {
	StatusCode int
	Header     http.Header
	Body       string
	Decoded    ParseErrorResponse
	URL        *url.URL
}

func newApiError(res *http.Response) *ApiError {
	data, _ := ioutil.ReadAll(res.Body)

	var parseErrorRes ParseErrorResponse
	_ = json.Unmarshal(data, &parseErrorRes)
	return &ApiError{
		StatusCode: res.StatusCode,
		Header:     res.Header,
		Body:       string(data),
		Decoded:    parseErrorRes,
		URL:        res.Request.URL,
	}
}

// ApiError supports the error interface
func (err ApiError) Error() string {
	return fmt.Sprintf("Request %s returned status %d, %s", err.URL, err.StatusCode, err.Body)
}
