package main

import "bitwarden-tui/internal/backend"

func main() {
	client, err := backend.New("cpktnwt9")
	if err != nil {
		panic(err)
	}

	err = client.CreateItem(&backend.Item{
		Name:   "Test item 1",
		Type:   1,
		Fields: []backend.Field{},
		Login: backend.Login{
			Username: "testusername",
			Password: "hunter2",
			Uris:     []backend.Uri{},
		},
	})
	if err != nil {
		panic(err)
	}
}
