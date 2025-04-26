package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"
)

type Config struct {
	Handle   string `json:"handle"`
	Password string `json:"password"`
	BaseURL  string `json:"baseURL"`
	DayCount int    `json:"dayCount"`
}

type Record struct {
	URI   string `json:"uri"`
	CID   string `json:"cid"`
	Value struct {
		CreatedAt string `json:"createdAt"`
		Text      string `json:"text"`
	} `json:"value"`
}

var config Config
var handle string
var password string
var baseURL string
var authToken string
var did string

func loadConfig() error {
	file, err := os.Open("config.json")
	if err != nil {
		return err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		return err
	}

	handle = config.Handle
	password = config.Password
	baseURL = config.BaseURL
	if baseURL == "" {
		return errors.New("baseURL is not set in config.json")
	}

	if config.DayCount == 0 {
		config.DayCount = 30 // Default to 30 days if not set
	}
	return nil
}

func login() error {
	url := baseURL + "/com.atproto.server.createSession"
	payload := map[string]string{
		"identifier": handle,
		"password":   password,
	}
	jsonPayload, _ := json.Marshal(payload)

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var responseBody bytes.Buffer
		responseBody.ReadFrom(resp.Body)
		return errors.New("failed to login: " + responseBody.String())
	}

	var result struct {
		AccessJwt string `json:"accessJwt"`
		Did       string `json:"did"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	authToken = result.AccessJwt
	did = result.Did
	return nil
}

func listPosts(cursor string) ([]Record, string, error) {
	url := baseURL + "/com.atproto.repo.listRecords"

	queryParams := "?repo=" + did + "&collection=app.bsky.feed.post"
	if cursor != "" {
		queryParams += "&cursor=" + cursor
	}
	url += queryParams

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+authToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var responseBody bytes.Buffer
		responseBody.ReadFrom(resp.Body)
		return nil, "", errors.New("failed to list posts: " + responseBody.String())
	}

	var result struct {
		Records []Record `json:"records"`
		Cursor  string   `json:"cursor"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, "", err
	}

	return result.Records, result.Cursor, nil
}

func deleteRecord(record Record) error {
	url := baseURL + "/com.atproto.repo.deleteRecord"
	payload := map[string]string{
		"uri": record.URI,
		"cid": record.CID,
	}
	jsonPayload, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonPayload))
	req.Header.Set("Authorization", "Bearer "+authToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errors.New("failed to delete record")
	}

	return nil
}

type App struct {
	Login        func() error
	ListPosts    func(cursor string) ([]Record, string, error)
	DeleteRecord func(record Record) error
}

func (app *App) Run() {
	if err := loadConfig(); err != nil {
		panic("Failed to load config: " + err.Error())
	}

	if err := app.Login(); err != nil {
		panic("Failed to login: " + err.Error())
	}

	cursor := ""
	deletedCount := 0
	skippedCount := 0 // Count of posts not deleted because they were newer
	for {
		records, nextCursor, err := app.ListPosts(cursor)
		if err != nil {
			panic("Failed to list posts: " + err.Error())
		}

		for _, record := range records {
			createdAt, _ := time.Parse(time.RFC3339, record.Value.CreatedAt)
			if time.Since(createdAt).Hours() > float64(config.DayCount*24) {
				if err := app.DeleteRecord(record); err != nil {
					fmt.Printf("Failed to delete record: %s\n", record.URI)
				} else {
					fmt.Printf("Deleted record: %s\n", record.URI)
					deletedCount++
				}
			} else {
				skippedCount++
			}
		}

		if nextCursor == "" {
			break
		}
		cursor = nextCursor
	}

	if deletedCount == 0 {
		fmt.Println("No posts were deleted.")
	} else {
		fmt.Printf("Total posts deleted: %d\n", deletedCount)
	}

	fmt.Printf("Total posts skipped (newer than %d days): %d\n", config.DayCount, skippedCount)
}

func main() {
	app := &App{
		Login:        login,
		ListPosts:    listPosts,
		DeleteRecord: deleteRecord,
	}
	app.Run()
}
