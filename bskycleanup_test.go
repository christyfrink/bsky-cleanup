package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestLogin(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/com.atproto.server.createSession" {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{
				"accessJwt": "mockToken",
				"did":       "mockDID",
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}

	server := httptest.NewServer(http.HandlerFunc(handler))
	defer server.Close()

	baseURL = server.URL
	if err := login(); err != nil {
		t.Fatalf("login failed: %v", err)
	}

	if authToken != "mockToken" || did != "mockDID" {
		t.Errorf("unexpected authToken or did: %s, %s", authToken, did)
	}
}

func TestListPosts(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/com.atproto.repo.listRecords" {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"records": []Record{
					{
						URI: "mockURI",
						CID: "mockCID",
						Value: struct {
							CreatedAt string `json:"createdAt"`
							Text      string `json:"text"`
						}{
							CreatedAt: time.Now().AddDate(0, 0, -31).Format(time.RFC3339),
							Text:      "Old post",
						},
					},
				},
				"cursor": "nextCursor",
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}

	server := httptest.NewServer(http.HandlerFunc(handler))
	defer server.Close()

	baseURL = server.URL
	records, cursor, err := listPosts("")
	if err != nil {
		t.Fatalf("listPosts failed: %v", err)
	}

	if len(records) != 1 || cursor != "nextCursor" {
		t.Errorf("unexpected records or cursor: %v, %s", records, cursor)
	}
}

func TestDeleteRecord(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/com.atproto.repo.deleteRecord" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}

	server := httptest.NewServer(http.HandlerFunc(handler))
	defer server.Close()

	baseURL = server.URL
	record := Record{
		URI: "mockURI",
		CID: "mockCID",
	}

	if err := deleteRecord(record); err != nil {
		t.Fatalf("deleteRecord failed: %v", err)
	}
}

func TestLoadConfig(t *testing.T) {
	// Backup and remove existing config.json if it exists
	if _, err := os.Stat("config.json"); err == nil {
		if err := os.Rename("config.json", "config.json.bak"); err != nil {
			t.Fatalf("failed to backup config.json: %v", err)
		}
		defer os.Rename("config.json.bak", "config.json") // Restore after test
	}

	// Test loading config without config.json
	err := loadConfig()
	if err == nil {
		t.Error("expected error when config.json is missing, got nil")
	}

	// Create a temporary config.json for testing
	configContent := `{
		"handle": "test-handle",
		"password": "test-password",
		"baseURL": "https://example.com"
	}`
	if err := os.WriteFile("config.json", []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to create temporary config.json: %v", err)
	}
	defer os.Remove("config.json") // Clean up after test

	// Test loading config with valid config.json
	err = loadConfig()
	if err != nil {
		t.Errorf("unexpected error when loading valid config.json: %v", err)
	}

	if config.Handle != "test-handle" || config.Password != "test-password" {
		t.Errorf("unexpected config values: %+v", config)
	}
}

func TestMainFeedback(t *testing.T) {
	app := &App{
		Login: func() error {
			return nil
		},
		ListPosts: func(cursor string) ([]Record, string, error) {
			if cursor == "" {
				return []Record{
					{
						URI: "mockURI1",
						CID: "mockCID1",
						Value: struct {
							CreatedAt string `json:"createdAt"`
							Text      string `json:"text"`
						}{
							CreatedAt: time.Now().AddDate(0, 0, -31).Format(time.RFC3339),
							Text:      "Old post",
						},
					},
				}, "nextCursor", nil
			}
			return nil, "", nil
		},
		DeleteRecord: func(record Record) error {
			if record.URI == "mockURI1" {
				return nil
			}
			return errors.New("failed to delete record")
		},
	}

	// Capture output using os.Pipe
	reader, writer, _ := os.Pipe()
	originalStdout := os.Stdout
	os.Stdout = writer

	output := &bytes.Buffer{}
	done := make(chan bool)
	go func() {
		_, _ = output.ReadFrom(reader)
		done <- true
	}()

	app.Run()

	// Restore original stdout and close the writer
	writer.Close()
	os.Stdout = originalStdout
	<-done

	// Check output
	if !strings.Contains(output.String(), "Deleted record: mockURI1") {
		t.Errorf("Expected feedback for deleted record, got: %s", output.String())
	}

	if !strings.Contains(output.String(), "Total posts deleted: 1") {
		t.Errorf("Expected total deleted count, got: %s", output.String())
	}
}

func TestDayCountSetting(t *testing.T) {
	app := &App{
		Login: func() error {
			return nil
		},
		ListPosts: func(cursor string) ([]Record, string, error) {
			if cursor == "" {
				return []Record{
					{
						URI: "mockURI1",
						CID: "mockCID1",
						Value: struct {
							CreatedAt string `json:"createdAt"`
							Text      string `json:"text"`
						}{
							CreatedAt: time.Now().AddDate(0, 0, -15).Format(time.RFC3339), // 15 days old
							Text:      "Recent post",
						},
					},
					{
						URI: "mockURI2",
						CID: "mockCID2",
						Value: struct {
							CreatedAt string `json:"createdAt"`
							Text      string `json:"text"`
						}{
							CreatedAt: time.Now().AddDate(0, 0, -45).Format(time.RFC3339), // 45 days old
							Text:      "Old post",
						},
					},
				}, "", nil
			}
			return nil, "", nil
		},
		DeleteRecord: func(record Record) error {
			if record.URI == "mockURI2" {
				return nil
			}
			return errors.New("failed to delete record")
		},
	}

	// Set dayCount to 30 for this test
	config.DayCount = 30

	// Capture output using os.Pipe
	reader, writer, _ := os.Pipe()
	originalStdout := os.Stdout
	os.Stdout = writer

	output := &bytes.Buffer{}
	done := make(chan bool)
	go func() {
		_, _ = output.ReadFrom(reader)
		done <- true
	}()

	app.Run()

	// Restore original stdout and close the writer
	writer.Close()
	os.Stdout = originalStdout
	<-done

	// Check output
	if strings.Contains(output.String(), "Deleted record: mockURI1") {
		t.Errorf("Expected recent post (mockURI1) to not be deleted, but it was.")
	}

	if !strings.Contains(output.String(), "Deleted record: mockURI2") {
		t.Errorf("Expected old post (mockURI2) to be deleted, but it was not.")
	}

	if !strings.Contains(output.String(), "Total posts deleted: 1") {
		t.Errorf("Expected total deleted count to be 1, got: %s", output.String())
	}
}
