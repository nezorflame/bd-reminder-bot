package slack

import (
	"encoding/json"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
	"github.com/valyala/fasthttp"
	ws "golang.org/x/net/websocket"
)

// Slack URL consts
const (
	methodGET      = "GET"
	methodPOST     = "POST"
	contentEncoded = "application/x-www-form-urlencoded; charset=utf-8"
	contentJSON    = "application/json; charset=utf-8"

	apiURL   = "https://api.slack.com/"
	startURL = "https://slack.com/api/rtm.start"
)

var (
	reqTimeout = 2 * time.Second
	wsDeadline = 100 * time.Millisecond
	retryCount = 3
)

var msgCounter uint64

// GetWSMessage receives a message from RTM API
func GetWSMessage(conn *ws.Conn) (m Message, err error) {
	if err = conn.SetReadDeadline(time.Now().Add(wsDeadline)); err != nil {
		return
	}
	err = ws.JSON.Receive(conn, &m)
	return
}

// SendWSMessage sends a message with RTM API
func SendWSMessage(conn *ws.Conn, m Message) error {
	m.ID = atomic.AddUint64(&msgCounter, 1)
	return ws.JSON.Send(conn, m)
}

// Connect creates a websocket-based Real Time API session and return the websocket
// and the ID of the (bot-)user whom the token belongs to.
func Connect(token string) (conn *ws.Conn, userID string, err error) {
	wsURL, userID, err := initWSConn(token)
	if err != nil {
		err = errors.Wrap(err, "unable to get websocket URL")
		return
	}

	if conn, err = ws.Dial(wsURL, "", apiURL); err != nil {
		err = errors.Wrap(err, "unable to call Slack's websocket")
	}

	return
}

func initWSConn(token string) (wsURL, userID string, err error) {
	var response struct {
		Ok    bool   `json:"ok"`
		Error string `json:"error"`
		URL   string `json:"url"`
		Self  struct {
			ID string `json:"id"`
		} `json:"self"`
	}

	params := map[string]string{"token": token}
	respBody, err := makeRequest(startURL, methodGET, contentEncoded, nil, params, nil)
	if err != nil {
		err = errors.Wrap(err, "unable to make GET request")
		return
	}

	if err = json.Unmarshal(respBody, &response); err != nil {
		err = errors.Wrap(err, "unable to unmarshal response")
		return
	}

	if !response.Ok {
		err = errors.Wrap(err, "request was unsuccessful")
		return
	}

	wsURL, userID = response.URL, response.Self.ID
	return
}

func makeRequest(url, method, contentType string, body []byte, params, headers map[string]string) ([]byte, error) {
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)
	req.Header.SetMethod(method)
	req.Header.SetContentType(contentType)
	if len(headers) > 0 {
		for k, v := range headers {
			req.Header.Set(k, v)
		}
	}

	req.SetRequestURI(url)
	if len(params) > 0 {
		for k, v := range params {
			req.URI().QueryArgs().Add(k, v)
		}
	}

	if body != nil {
		req.SetBody(body)
	}

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)
	var (
		err   error
		count int
	)
	for count = 0; count < retryCount; count++ {
		if err = fasthttp.DoTimeout(req, resp, reqTimeout); err == nil {
			code := resp.StatusCode()
			if code == fasthttp.StatusOK {
				break
			}
			err = errors.Errorf("%d: %s", code, fasthttp.StatusMessage(code))
		}
	}
	if err != nil {
		return nil, errors.Wrapf(err, "request failed after %d retries", count)
	}

	respBody := make([]byte, len(resp.Body()))
	copy(respBody, resp.Body())
	return respBody, nil
}
