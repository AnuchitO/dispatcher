package dispatcher

import (
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"path"
	"reflect"
	"sync"
)

// HTTPClient type
type HTTPClient struct {
	URL         string
	Client      *http.Client
	ResultField string

	once sync.Once
}

func (c *HTTPClient) init() {
	c.once.Do(func() {
		if c.Client == nil {
			c.Client = http.DefaultClient
		}
		if c.ResultField == "" {
			c.ResultField = "Result"
		}
	})
}

func (c *HTTPClient) dispatch(ctx context.Context, msg Message) error {
	buf := getBytes()
	defer putBytes(buf)

	refMsg := reflect.ValueOf(msg)
	refResult := refMsg.Elem().FieldByName(c.ResultField)

	err := json.NewEncoder(buf).Encode(msg)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, path.Join(c.URL, msgName(msg)), buf)
	if err != nil {
		return err
	}

	resp, err := c.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	defer io.Copy(ioutil.Discard, resp.Body)

	err = json.NewDecoder(resp.Body).Decode(refResult.Interface())
	if err != nil {
		return err
	}

	return nil
}

// Dispatch dispatchs message thought http
func (c *HTTPClient) Dispatch(ctx context.Context, msgs ...Message) error {
	c.init()

	// TODO: bulk dispatch ?
	var err error
	for _, msg := range msgs {
		switch p := msg.(type) {
		case []Message:
			err = c.Dispatch(ctx, p...)
		case Messages:
			err = c.Dispatch(ctx, p...)
		default:
			err = c.dispatch(ctx, msg)
		}
		if err != nil {
			return err
		}
	}
	return nil
}
