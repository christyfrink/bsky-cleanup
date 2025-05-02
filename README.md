# Bluesky Cleanup Script

[![Run Tests](https://github.com/stephenyeargin/bskycleanup/actions/workflows/ci.yml/badge.svg)](https://github.com/stephenyeargin/bskycleanup/actions/workflows/ci.yml)

This script is designed to clean up Bluesky posts older than 30 days (by default). It authenticates with your Bluesky account and deletes posts that meet the criteria.

## Configuration

1. Create a `config.json` file in the root directory of the project with the following structure:

```json
{
  "handle": "your-handle",
  "password": "your-app-password",
  "baseURL": "https://bsky.social/xrpc",
  "dayCount": 30
}
```

- Replace `your-handle` with your Bluesky handle.
- Replace `your-app-password` with a Bluesky app password (Generate from Bluesky settings > Privacy and Security > App Passwords).
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

## Setting Up GitHub Actions for Daily Cleanup

To automate the cleanup script to run daily, follow these steps:

1. **Add Repository Secrets**:
   - Go to your GitHub repository.
   - Navigate to `Settings` > `Secrets and variables` > `Actions`.
   - Add the following secrets:
     - `BSKY_BASE_URL`: The base URL for the Bluesky API (e.g., `https://bsky.social/xrpc`).
     - `BSKY_HANDLE`: Your Bluesky handle.
     - `BSKY_PASSWORD`: Your Bluesky app password.
     - `BSKY_DAY_COUNT`: Number of days to keep.

2. **Verify the Workflow File**:
   - Ensure the `.github/workflows/daily-cleanup.yml` file exists in your repository.
   - This file is already configured to:
     - Clone the repository.
     - Create a `config.json` file using the repository secrets.
     - Run the cleanup script daily at midnight UTC.

3. **Enable GitHub Actions**:
   - Ensure GitHub Actions is enabled for your repository.
   - The workflow will automatically trigger daily based on the schedule defined in the `daily-cleanup.yml` file.

4. **Monitor Workflow Runs**:
   - Go to the `Actions` tab in your GitHub repository.
   - Check the `Daily Cleanup` workflow to monitor its execution and logs.

By setting this up, the cleanup script will run daily without manual intervention, ensuring your Bluesky posts are managed automatically.
