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

type ParseClient struct {
	parseAppId   string
	parseApiKey  string
	client       *http.Client
	requestQueue chan request
}

type request struct {
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

type CreateResponse struct {
	CreatedAt string `json:"createdAt"`
	ObjectId  string `json:"objectId"`
}

type UpdateResponse struct {
	UpdatedAt string `json:"updatedAt"`
}

type deleteResponse struct {
	dummy string
}

func Client(AppId string, AppKey string, c appengine.Context) *ParseClient {
	queue := make(chan request)
	client := &ParseClient{}
	if c != nil {
		client = &ParseClient{
			parseAppId:   AppId,
			parseApiKey:  AppKey,
			client:       urlfetch.Client(c),
			requestQueue: queue,
		}
	} else {
		client = &ParseClient{
			parseAppId:   AppId,
			parseApiKey:  AppKey,
			client:       http.DefaultClient,
			requestQueue: queue,
		}
	}
	go client.throttledQuery()
	return client
}

func (c ParseClient) CreateObject(className string, reqData io.Reader) (*CreateResponse, error) {
	response_ch := make(chan response)
	var cr CreateResponse
	c.requestQueue <- request{_POST, parseUrl + className, nil, reqData, &cr, response_ch}
	rc := (<-response_ch)
	return rc.resData.(*CreateResponse), rc.err
}

func (c ParseClient) RetrieveObject(className string, objectId string, v url.Values, resData interface{}) (interface{}, error) {
	response_ch := make(chan response)
	c.requestQueue <- request{_GET, parseUrl + className + "/" + objectId, v, nil, resData, response_ch}
	rc := (<-response_ch)
	return rc.resData, rc.err
}

func (c ParseClient) RetrieveObjects(className string, v url.Values, resData interface{}) (interface{}, error) {
	response_ch := make(chan response)
	c.requestQueue <- request{_GET, parseUrl + className, v, nil, resData, response_ch}
	rc := (<-response_ch)
	return rc.resData, rc.err
}

func (c ParseClient) UpdateObject(className string, objectId string, reqData io.Reader) (*UpdateResponse, error) {
	response_ch := make(chan response)
	var ur UpdateResponse
	c.requestQueue <- request{_PUT, parseUrl + className + "/" + objectId, nil, reqData, &ur, response_ch}
	rc := (<-response_ch)
	return rc.resData.(*UpdateResponse), rc.err
}

func (c ParseClient) DeleteObject(className string, objectId string) error {
	response_ch := make(chan response)
	var dr deleteResponse
	c.requestQueue <- request{_DELETE, parseUrl + className + "/" + objectId, nil, nil, &dr, response_ch}
	return (<-response_ch).err
}

func (c *ParseClient) throttledQuery() {
	for r := range c.requestQueue {
		response_ch := r.response_ch
		resData, err := c.execHttpRequest(r.method, r.url, r.form, r.reqData, r.resData)
		if err != nil {
			// TODO Check rate-limit and retry
		}
		response_ch <- response{resData, err}
	}
}

func (c ParseClient) execHttpRequest(method int, url string, form url.Values, reqData io.Reader, resData interface{}) (interface{}, error) {
	var methodStr string
	switch method {
	case _POST:
		methodStr = "POST"
	case _GET:
		methodStr = "GET"
	case _PUT:
		methodStr = "PUT"
	case _DELETE:
		methodStr = "DELETE"
	default:
		return nil, fmt.Errorf("HTTP method not yet supported")
	}
	query := form.Encode()
	req, _ := http.NewRequest(methodStr, url+"?"+query, reqData)
	req.Header.Set(parseAppIdHeader, c.parseAppId)
	req.Header.Add(parseApiKeyHeader, c.parseApiKey)
	req.Header.Add(contentTypeHeader, contentType)
	res, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	return decodeResponse(res, resData)
}

func decodeResponse(res *http.Response, resData interface{}) (interface{}, error) {
	fmt.Printf("decode: %s\n", resData)
	if (res.StatusCode != 200) && (res.StatusCode != 201) {
		return nil, newApiError(res)
	}
	err := json.NewDecoder(res.Body).Decode(resData)
	return resData, err
}

type ErrorResponse struct {
	Code  int    `json:"code"`
	Error string `json:"error"`
}

// ApiError is an error implementation
type ApiError struct {
	StatusCode int
	Header     http.Header
	Body       string
	Decoded    ErrorResponse
	URL        *url.URL
}

func newApiError(res *http.Response) *ApiError {
	data, _ := ioutil.ReadAll(res.Body)

	var errorResponse ErrorResponse
	_ = json.Unmarshal(data, &errorResponse)
	return &ApiError{
		StatusCode: res.StatusCode,
		Header:     res.Header,
		Body:       string(data),
		Decoded:    errorResponse,
		URL:        res.Request.URL,
	}
}

// ApiError supports the error interface
func (err ApiError) Error() string {
	return fmt.Sprintf("Request %s returned status %d, %s", err.URL, err.StatusCode, err.Body)
}
