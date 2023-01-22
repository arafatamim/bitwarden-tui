package backend

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
)

type Client struct {
	SessionKey string
}

type Uri struct {
	Uri string `json:"uri"`
}

type Login struct {
	Uris     []Uri  `json:"uris"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type Item struct {
	OrganizationId *string `json:"organizationId"`
	CollectionIds  []int   `json:"collectionIds"`
	FolderId       *string `json:"folderId"`
	Type           int     `json:"type"`
	Id             string  `json:"id,omitempty"`
	Name           string  `json:"name"`
	Notes          *string `json:"notes"`
	Favorite       bool    `json:"favorite"`
	Fields         []Field `json:"fields"`
	Login          Login   `json:"login"`
	Reprompt       uint8   `json:"reprompt"`
}

type Folder struct {
	Object string `json:"folder"`
	Id     string `json:"id"`
	Name   string `json:"name"`
}

type FilterOptions struct {
	Search string
	Url    string
}

type Field struct {
	Name     string `json:"name"`
	Value    string `json:"value"`
	Type     uint8  `json:"type"` // text - 0, hidden - 1, boolean - 2
	LinkedId int    `json:"linkedId"`
}

func pipeToFifo(s string) error {
	file, err := os.OpenFile("stdpipe", os.O_APPEND|os.O_RDWR, os.ModeNamedPipe)
	if err != nil {
		return err
	}
	file.WriteString(s)

	return nil
}

func (c *Client) exec(args ...string) ([]byte, error) {
	cmd := exec.Command("bw", args...)

	// stderr := bytes.NewBufferString("")
	// cmd.Stderr = stderr

	output, err := cmd.Output()
	if err != nil {
		if command := err.(*exec.ExitError); command.Stderr != nil {
			return output, errors.New(string(command.Stderr))
		}
	}

	return output, err
}

func New(password string) (*Client, error) {
	client := &Client{}
	key, err := client.exec("unlock", "--raw", password)
	if err != nil {
		return nil, err
	}
	client.SessionKey = string(key)
	err = os.Setenv("BW_SESSION", client.SessionKey)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func NewFromSessionKey(key string) (*Client, error) {
	if len(key) == 0 {
		return nil, errors.New("Session key is empty!")
	}
	client := &Client{
		SessionKey: key,
	}
	err := os.Setenv("BW_SESSION", key)
	return client, err
}

func (c *Client) GetItems(filter FilterOptions) ([]Item, error) {
	output, err := c.exec("list", "items", "--search", filter.Search, "--url", filter.Url)
	if err != nil {
		return nil, err
	}
	var items []Item
	err = json.Unmarshal(output, &items)
	if err != nil {
		return nil, err
	}
	items = FilterItems(items, func(i Item) bool {
		return i.Type == 1
	})
	return items, nil
}

func (c *Client) GetItem(id string) (*Item, error) {
	output, err := c.exec("get", "item", id)
	if err != nil {
		return nil, err
	}
	var item *Item
	err = json.Unmarshal(output, &item)
	if err != nil {
		return nil, err
	}
	return item, nil
}

func (c *Client) GetFolder(id string) (*Folder, error) {
	output, err := c.exec("get", "folder", id)
	if err != nil {
		return nil, err
	}
	var folder *Folder
	err = json.Unmarshal(output, &folder)
	if err != nil {
		return nil, err
	}
	return folder, nil
}

func (c *Client) GetFolders() ([]Folder, error) {
	output, err := c.exec("get", "folders")
	if err != nil {
		return nil, err
	}
	var folders []Folder
	err = json.Unmarshal(output, &folders)
	if err != nil {
		return nil, err
	}
	return folders, nil
}

func (c *Client) Sync() error {
	_, err := c.exec("sync")
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) CreateItem(item *Item) error {
	j, err := json.Marshal(item)
	if err != nil {
		return err
	}
	fmt.Println(string(j))
	b := bytes.NewBufferString("")
	enc := base64.NewEncoder(base64.StdEncoding, b)
	enc.Write(j)
	err = enc.Close()

	_, err = c.exec("create", "item", b.String())
	if err != nil {
		return err
	}

	return nil
}

func FilterItems(vs []Item, f func(Item) bool) []Item {
	filtered := make([]Item, 0)
	for _, v := range vs {
		if f(v) {
			filtered = append(filtered, v)
		}
	}
	return filtered
}

func MapFields(vs []Field, f func(Field) string) []string {
	mapped := make([]string, 0)
	for _, v := range vs {
		mapped = append(mapped, f(v))
	}
	return mapped
}
