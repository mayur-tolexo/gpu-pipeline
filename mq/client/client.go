package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

type Client struct {
	baseURL string
	http    *http.Client
}

func New(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		http:    &http.Client{},
	}
}

func (c *Client) CreateTopic(name string, partitions int) error {
	body := map[string]interface{}{
		"name":       name,
		"partitions": partitions,
	}

	data, _ := json.Marshal(body)

	resp, err := c.http.Post(c.baseURL+"/topics", "application/json", bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("failed to create topic: %s", resp.Status)
	}

	return nil
}

func (c *Client) Publish(topic string, msg Message) (int, int, error) {
	data, _ := json.Marshal(msg)

	resp, err := c.http.Post(
		fmt.Sprintf("%s/topics/%s/publish", c.baseURL, topic),
		"application/json",
		bytes.NewReader(data),
	)
	if err != nil {
		return 0, 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, 0, fmt.Errorf("publish failed: %s", resp.Status)
	}

	var res PublishResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return 0, 0, err
	}

	return res.Partition, res.Offset, nil
}

func (c *Client) Consume(topic, group string, partition, batch int) (*ConsumeResponse, error) {
	u := fmt.Sprintf("%s/topics/%s/consume", c.baseURL, topic)

	params := url.Values{}
	params.Set("group", group)
	params.Set("partition", fmt.Sprintf("%d", partition))
	params.Set("batch", fmt.Sprintf("%d", batch))

	fullURL := u + "?" + params.Encode()

	resp, err := c.http.Get(fullURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("consume failed: %s", resp.Status)
	}

	var res ConsumeResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}

	return &res, nil
}

func (c *Client) Ack(topic string, req AckRequest) error {
	data, _ := json.Marshal(req)

	resp, err := c.http.Post(
		fmt.Sprintf("%s/topics/%s/ack", c.baseURL, topic),
		"application/json",
		bytes.NewReader(data),
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ack failed: %s", resp.Status)
	}

	return nil
}
