# multiUploader

> A cross-platform GUI application for uploading files to multiple file hosting services with built-in retry mechanism and progress tracking.

[![Go Version](https://img.shields.io/badge/Go-1.24%2B-blue)](https://golang.org/)
[![Fyne](https://img.shields.io/badge/Fyne-v2.7.1-purple)](https://fyne.io/)
[![Platform](https://img.shields.io/badge/platform-macOS%20%7C%20Linux%20%7C%20Windows-lightgrey)](https://github.com/fyne-io/fyne)

## Features

- ✅ **Cross-platform GUI** - Works on macOS, Linux, and Windows
- ✅ **4 File Hosting Providers** - Rootz, DataVaults, AkiraBox, FileKeeper
- ✅ **Chunked Upload** - Efficient batch uploading for large files
- ✅ **Real-time Progress** - Live progress bar, speed, and ETA
- ✅ **Automatic Retry** - Built-in exponential backoff for network failures
- ✅ **User-friendly Errors** - Clear error messages with actionable hints
- ✅ **Structured Logging** - JSON logs for bug reports
- ✅ **Connection Pooling** - Optimized HTTP client for better performance

## Supported Providers

| Provider | Status | API Documentation |
|----------|--------|-------------------|
| [Rootz.so](https://rootz.so) | ✅ Ready | [API Docs](https://www.rootz.so/docs) |
| [DataVaults.co](https://datavaults.co) | ✅ Ready | [API Docs](https://datavaults.co/pages/api) |
| [AkiraBox.com](https://akirabox.com) | ✅ Ready | [API Docs](https://akirabox.com/api) |
| [FileKeeper.net](https://filekeeper.net) | ✅ Ready | [API Docs](https://datanodes.docs.apiary.io/) |

## Installation

### Download Pre-built Binary

*Releases coming soon*

### Build from Source

**Requirements:**
- Go 1.24 or higher
- Internet connection (for dependencies)

**Build steps:**

```bash
# Clone the repository
git clone https://github.com/yourusername/multiUploader.git
cd multiUploader

# Download dependencies
go mod download

# Build the application
go build -o multiUploader main.go

# Run
./multiUploader
```

**Development mode:**

```bash
go run main.go
```

## Getting Started

### 1. Obtain API Keys

You need API keys for the providers you want to use:

#### Rootz.so
1. Visit https://www.rootz.so/
2. Create an account or log in
3. Navigate to Settings → API
4. Generate a new API key

#### DataVaults.co
1. Visit https://datavaults.co/
2. Create an account or log in
3. Go to Account Settings → API
4. Copy your API key

#### AkiraBox.com
1. Visit https://akirabox.com/
2. Sign up or log in
3. Navigate to Profile → API Keys
4. Generate a new key

#### FileKeeper.net
1. Visit https://filekeeper.net/
2. Register or log in
3. Go to Settings → API Access
4. Create an API key

### 2. Configure Providers

1. Launch multiUploader
2. Go to **Settings** tab
3. For each provider:
   - Toggle **Enable** checkbox
   - Paste your **API Key**
   - (Optional) Set custom chunk size
4. Click **Save Settings**

### 3. Upload Files

1. Go to **Upload** tab
2. Select a provider from the dropdown
3. Click **Select File** and choose a file (resizable file picker!)
4. Click **Upload**
5. Watch real-time progress:
   - Progress bar with percentage
   - Upload speed (B/s, KB/s, MB/s)
   - Uploaded / Total size
   - Estimated time remaining (ETA)
6. After upload completes, copy URLs from the result dialog

**Tip:** You can cancel an upload anytime by clicking **Cancel**.

## Configuration

### Settings Location

Settings are stored in platform-specific locations:

| Platform | Location |
|----------|----------|
| **macOS** | `~/Library/Preferences/multiUploader/` |
| **Linux** | `~/.config/multiUploader/` |
| **Windows** | `%APPDATA%\multiUploader\` |

### Global Settings

- **Theme** - Light, Dark, or Auto (system default)
- **Language** - English, Russian, or Auto (system default)

### Provider Settings

For each provider:
- **Enable/Disable** - Toggle provider availability
- **API Key** - Your authentication key

## Logs and Debugging

### Log Location

Logs are stored in platform-specific locations:

| Platform | Location |
|----------|----------|
| **macOS** | `~/Library/Logs/multiUploader/app.log` |
| **Linux** | `~/.local/share/multiUploader/logs/app.log` |
| **Windows** | `%LOCALAPPDATA%\multiUploader\logs\app.log` |

### Accessing Logs

**Via Menu:**
1. Click **File** → **Open Logs Folder**
2. The log directory will open in your file manager

**Log Format:**
- JSON structured logs
- Only ERROR level (for bug reports)
- Includes: timestamp, error message, provider, filename, file size
- Automatic rotation at 5 MB (keeps 1 backup file)

**Example log entry:**
```json
{
  "time": "2026-01-07T22:13:01.206Z",
  "level": "ERROR",
  "source": {"function": "upload_tab.finishUpload", "file": "upload_tab.go", "line": 350},
  "msg": "Upload failed",
  "provider": "Rootz",
  "filename": "document.pdf",
  "filesize": 1048576,
  "error": "connection timeout"
}
```

## Troubleshooting

### Upload Fails with "Connection Timeout"

**Cause:** Network connectivity issues or server is slow to respond.

**Solution:**
- Check your internet connection
- The application will automatically retry (up to 3 times with exponential backoff)
- Try again in a few minutes

### "Invalid API Key" Error

**Cause:** The API key is incorrect or expired.

**Solution:**
1. Go to **Settings** tab
2. Verify the API key is correct
3. Generate a new key from the provider's website if needed
4. **Save Settings** after updating

### File Picker Window is Too Small

**Fixed!** The file picker now opens at 800x600 for better navigation.

### Upload Stalls at 0%

**Possible causes:**
- File is locked by another program
- Insufficient permissions
- Network connection dropped

**Solution:**
- Close other programs using the file
- Check file permissions
- Check **File** → **Open Logs Folder** for detailed error

### Provider Shows "Disabled" in Dropdown

**Cause:** Provider is not enabled in Settings.

**Solution:**
1. Go to **Settings** tab
2. Find the provider
3. Check the **Enable** checkbox
4. Enter API key if not set
5. Click **Save Settings**
6. Return to **Upload** tab

## FAQ

**Q: Which provider should I use?**

A: All providers have different features and limits. Try each one to see which works best for your needs.

**Q: Can I upload multiple files at once?**

A: Not yet. Currently, you can upload one file at a time. Batch upload is planned for future versions.

**Q: What's the maximum file size?**

A: Depends on the provider. Each provider has different limits. Check their documentation for details.

**Q: Is my API key stored securely?**

A: API keys are stored in your system's application preferences folder with standard OS permissions. They are not encrypted.

**Q: Does this work offline?**

A: No, internet connection is required to upload files to hosting providers.

**Q: Can I see upload history?**

A: Not yet. Upload history is planned for future versions.

## Advanced Features

### Retry Mechanism

The application includes smart retry logic:
- **Automatic retries** for temporary network errors (timeout, DNS failure, connection refused)
- **Exponential backoff** - waits longer between each retry (500ms → 1s → 2s → 4s...)
- **Max 3 retries** with 5-minute total timeout
- **Only for safe operations** - GET, PUT, DELETE (not POST for safety)
- **Retriable HTTP status codes** - 408, 429, 500, 502, 503, 504

### Connection Pooling

HTTP connections are reused for better performance:
- **100 max idle connections** across all providers
- **10 connections per host** for parallel uploads
- **90-second keep-alive** to avoid reconnections
- **Shared HTTP client** - one client for all providers

This is especially beneficial for Rootz.so which makes 100+ requests for large files!

## Development

### Architecture

The project follows a multi-layered architecture:

```
┌─────────────────────────────────────┐
│         UI Layer (Fyne)             │
│  ┌──────────┐      ┌─────────────┐ │
│  │ Upload   │      │  Settings   │ │
│  │   Tab    │      │     Tab     │ │
│  └──────────┘      └─────────────┘ │
└──────────┬──────────────────────────┘
           │
┌──────────▼──────────────────────────┐
│    Business Logic Layer             │
│  ┌─────────────────────────────┐   │
│  │   Provider Interface        │   │
│  ├─────────────────────────────┤   │
│  │ Rootz │ DataVaults │ ...    │   │
│  └─────────────────────────────┘   │
└──────────┬──────────────────────────┘
           │
┌──────────▼──────────────────────────┐
│  Infrastructure Layer               │
│  ┌──────────┐  ┌──────────────┐    │
│  │  Config  │  │ HTTP Client  │    │
│  │  Manager │  │  with Retry  │    │
│  └──────────┘  └──────────────┘    │
└─────────────────────────────────────┘
```

### Adding a New Provider

1. Create a new file: `internal/providers/newprovider.go`
2. Implement the `Provider` interface:

```go
type Provider interface {
    Name() string
    Upload(ctx context.Context, file io.ReadSeeker, filename string,
           fileSize int64, progress chan<- UploadProgress) (*UploadResult, error)
    RequiresAuth() bool
    ValidateAPIKey(apiKey string) error
    DefaultChunkSize() int64
}
```

3. Register in `main.go`:

```go
multiApp.RegisterProviderFactory("YourProvider", func(apiKey string) providers.Provider {
    return providers.NewYourProvider(apiKey)
})
```

4. Use `httpclient.Default()` or `httpclient.LongLived()` for HTTP requests
5. Send progress updates through the channel
6. Use `logging.ErrorWithError()` to log errors

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run specific test
go test -run TestConfigManager ./internal/config
```

### Code Quality

**Test Coverage:**
- `internal/config`: 100% ✅
- `internal/providers`: 9.8% (critical utilities covered)
- `internal/ui`: 0% (GUI testing not required)

## Technology Stack

- **Language:** Go 1.24+
- **GUI Framework:** [Fyne v2.7.1](https://fyne.io/)
- **HTTP Retry:** [backoff/v4](https://github.com/cenkalti/backoff)
- **Logging:** Go standard library `log/slog`
- **Configuration:** Fyne Preferences API

## Contributing

Contributions are welcome! Please:

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## License

MIT License - see LICENSE file for details

## Acknowledgments

- Built with [Fyne](https://fyne.io/) GUI toolkit
- Retry logic powered by [cenkalti/backoff](https://github.com/cenkalti/backoff)
- Icons from Fyne theme

## Support

- **Issues:** [GitHub Issues](https://github.com/VerTox/multiUploader/issues)
- **Logs:** Check `File → Open Logs Folder` for debugging

---

Made with ❤️ using Go and Fyne
