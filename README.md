# Bluesky Cleanup Script

This script is designed to clean up Bluesky posts older than 30 days (by default). It authenticates with your Bluesky account and deletes posts that meet the criteria.

## Configuration

1. Create a `config.json` file in the root directory of the project with the following structure:

```json
{
  "handle": "your-handle",
  "password": "your-password",
  "baseURL": "https://bsky.social/xrpc",
  "dayCount": 30
}
```

- Replace `your-handle` with your Bluesky handle.
- Replace `your-password` with your Bluesky password.
- Ensure `baseURL` is set to the correct API endpoint, such as `https://bsky.social/xrpc`.

## Usage

1. Ensure you have Go installed on your system. You can download it from [golang.org](https://golang.org/).
2. Open a terminal and navigate to the project directory:

```bash
cd /Users/stephen/Sites/bskycleanup
```

3. Run the script:

```bash
go run bskycleanup.go
```

The script will authenticate with your Bluesky account and delete posts older than 30 days.

## Running the Test Suite

1. To run the test suite, use the following command:

```bash
go test ./...
```

This will execute all the tests in the project and display the results.

## Notes

- Ensure your `config.json` file is not shared or committed to version control to protect your credentials.
- The script uses the Bluesky API, so ensure your account has the necessary permissions to delete posts.