package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
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

func listRecords(cursor, collection string) ([]Record, string, error) {
	url := baseURL + "/com.atproto.repo.listRecords"

	queryParams := "?repo=" + did + "&collection=app.bsky.feed." + collection
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
		return nil, "", errors.New("failed to list records: " + responseBody.String())
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

func deleteRecord(record Record, collection string) error {
	url := baseURL + "/com.atproto.repo.deleteRecord"

	// Extract rkey from the record URI
	uriParts := strings.Split(record.URI, "/")
	rkey := uriParts[len(uriParts)-1]

	payload := map[string]string{
		"repo":       did,
		"collection": "app.bsky.feed." + collection,
		"rkey":       rkey,
		"cid":        record.CID,
	}
	jsonPayload, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonPayload))
	req.Header.Set("Authorization", "Bearer "+authToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var responseBody bytes.Buffer
		responseBody.ReadFrom(resp.Body)
		return fmt.Errorf("Failed to delete %s: %s (status: %d, response: %s)", collection, record.URI, resp.StatusCode, responseBody.String())
	}

	return nil
}

var flagsParsed bool

type AppFlags struct {
	OnlyPosts    bool
	OnlyReposts  bool
	OnlyLikes    bool
	IncludeLikes bool
}

func getFlags() AppFlags {
	if !flagsParsed {
		flag.Bool("only-posts", false, "delete posts")
		flag.Bool("only-reposts", false, "delete reposts")
		flag.Bool("only-likes", false, "delete likes")
		flag.Bool("include-likes", false, "include likes with posts and reposts")

		flag.Parse()
		flagsParsed = true
	}

	return AppFlags{
		OnlyPosts:    flag.Lookup("only-posts").Value.(flag.Getter).Get().(bool),
		OnlyReposts:  flag.Lookup("only-reposts").Value.(flag.Getter).Get().(bool),
		OnlyLikes:    flag.Lookup("only-likes").Value.(flag.Getter).Get().(bool),
		IncludeLikes: flag.Lookup("include-likes").Value.(flag.Getter).Get().(bool),
	}
}

func setTypes(flags AppFlags) []string {
	var recordTypes []string
	defaultTypes := []string{"post", "repost"}

	switch {
	case flags.OnlyPosts:
		recordTypes = []string{"post"}
	case flags.OnlyReposts:
		recordTypes = []string{"repost"}
	case flags.OnlyLikes:
		recordTypes = []string{"like"}
	case flags.IncludeLikes:
		recordTypes = append(defaultTypes, "like")
	default:
		recordTypes = defaultTypes
	}

	return recordTypes
}

type App struct {
	Login        func() error
	AppFlags     AppFlags
	ListRecords  func(cursor string, collection string) ([]Record, string, error)
	DeleteRecord func(record Record, collection string) error
}

func (app *App) Run() {
	if err := loadConfig(); err != nil {
		panic("Failed to load config: " + err.Error())
	}

	if err := app.Login(); err != nil {
		panic("Failed to login: " + err.Error())
	}

	recordTypes := setTypes(app.AppFlags)

	for _, recordType := range recordTypes {
		types := recordType + "s"
		cursor := ""
		deletedCount := 0
		recordCount := 0
		for {
			records, nextCursor, err := app.ListRecords(cursor, recordType)
			if err != nil {
				panic("Failed to list " + types + ": " + err.Error())
			}

			recordCount += len(records)
			for _, record := range records {

				createdAt, _ := time.Parse(time.RFC3339, record.Value.CreatedAt)
				if time.Since(createdAt).Hours() > float64(config.DayCount*24) {
					if err := app.DeleteRecord(record, recordType); err != nil {
						fmt.Print(err)
						fmt.Printf("Failed to delete %s: %s\n", recordType, record.URI)
					} else {
						deletedCount++
						fmt.Printf("Deleted %s: %s\n", recordType, record.URI)
					}
				}
			}

			if nextCursor == "" {
				if deletedCount == 0 {
					fmt.Printf("No %s were deleted.\n", types)
				} else {
					fmt.Printf("Total %s deleted: %d\n", types, deletedCount)
				}
				fmt.Printf("Total %s skipped (newer than %d days): %d\n", recordType+"s", config.DayCount, recordCount-deletedCount)
				break
			}

			cursor = nextCursor
		}
	}
}

func main() {
	appFlags := getFlags()

	app := &App{
		Login:        login,
		AppFlags:     appFlags,
		ListRecords:  listRecords,
		DeleteRecord: deleteRecord,
	}
	app.Run()
}
