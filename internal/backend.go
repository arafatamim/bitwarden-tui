package backend

import (
	"encoding/json"
	"os"
	"os/exec"
)

type Context struct {
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
	Id       string  `json:"id"`
	FolderId string  `json:"folderId"`
	Type     int     `json:"type"`
	Name     string  `json:"name"`
	Notes    string  `json:"notes,omitempty"`
	Favorite bool    `json:"favorite"`
	Fields   []Field `json:"fields"`
	Login    Login   `json:"login"`
}

type Folder struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

type FilterOptions struct {
	Search string
	Url    string
}

type Field struct {
	Name     string `json:"name"`
	Value    string `json:"value"`
	Type     int8   `json:"type"`   // text - 0, hidden - 1, boolean - 2
	LinkedId int    `json:"linkedId"`
}

func (c *Context) exec(args ...string) ([]byte, error) {
	cmd := exec.Command("bw", args...)
	output, err := cmd.Output()
	return output, err
}

func InitializeClient(password string) (*Context, error) {
	ctx := &Context{}
	key, err := ctx.exec("unlock", "--raw", password)
	if err != nil {
		return nil, err
	}
	ctx.SessionKey = string(key)
	err = os.Setenv("BW_SESSION", ctx.SessionKey)
	if err != nil {
		return nil, err
	}
	return ctx, nil
}

func (c *Context) GetItems(filter FilterOptions) ([]Item, error) {
	output, err := c.exec("list", "items", "--search", filter.Search, "--url", filter.Url)
	if err != nil {
		return nil, err
	}
	var items []Item
	err = json.Unmarshal(output, &items)
	if err != nil {
		return nil, err
	}
	items = Filter(items, func(i Item) bool {
		return i.Type == 1
	})
	return items, nil
}

func (c *Context) GetItem(id string) (*Item, error) {
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

func (c *Context) GetFolder(id string) (*Folder, error) {
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

func (c *Context) Sync() error {
	_, err := c.exec("sync")
	if err != nil {
		return err
	}
	return nil
}

func Filter(vs []Item, f func(Item) bool) []Item {
	filtered := make([]Item, 0)
	for _, v := range vs {
		if f(v) {
			filtered = append(filtered, v)
		}
	}
	return filtered
}

func Map(vs []Field, f func(Field) string) []string {
	mapped := make([]string, 0)
	for _, v := range vs {
		mapped = append(mapped, f(v))
	}
	return mapped
}
