## Lightweight, high-performance HTTP file downloader

### Designed for developers, data scientists, and machine learning practitioners who need to handle large datasets, models, or software packages. With support for concurrent downloads, rate limiting, and resumable transfers.

## Key Features

- **Concurrent Downloads**: Download multiple files simultaneously to save time.
- **Rate Limiting**: Control bandwidth usage with precise speed limits.
- **Resumable Downloads**: Resume interrupted downloads seamlessly.
- **Custom Headers**: Add custom HTTP headers for APIs or secure downloads.
- **Proxy Support**: Configure HTTP or HTTPS proxies with environment variables.

## Installation and Usage

```bash
gograb [--header <key:value> [--header <key:value>]] [[rate limit:]url...]
```

### Arguments

| Argument     | Description                                                       |
| ------------ | ----------------------------------------------------------------- |
| `--header`   | Specify HTTP headers in the format `"key:value"`.                 |
| `rate limit` | Limit download speed, specified in KB (e.g., `200:` for 200KB/s). |
| `url...`     | One or more URLs to download.                                     |

#### Concurrent Downloads

Download multiple files concurrently to optimize time:

```bash
gograb https://example.com/file1.zip https://example.com/file2.zip
```

Example output:

| File Name   | Size  | Progress           | ETA     | Speed     |
| ----------- | ----- | ------------------ | ------- | --------- |
| `file1.zip` | 1.5GB | `[====>         ]` | `5m12s` | `4.3MB/s` |
| `file2.zip` | 750MB | `[=====>        ]` | `2m45s` | `3.1MB/s` |

``

#### Rate-Limited Downloads

Control your bandwidth by setting a download speed limit (e.g., 200KB/s):

```bash
gograb 200:https://example.com/largefile.iso
```

Example output:

| File Name       | Size  | Speed Limit | ETA     |
| --------------- | ----- | ----------- | ------- |
| `largefile.iso` | 4.7GB | `200KB/s`   | `6h30m` |

### Resumable Downloads

Start downloading a large file, interrupt it, and resume from where it left off:

1. Start the download using `gograb https://example.com/largefile.tar.gz`
2. Interrupt the download (e.g., pressing Ctrl+C)
3. Resume the download using `gograb https://example.com/largefile.tar.gz`

| File Name          | Size | Progress Before Interrupt | Progress After Resume  | ETA     | Speed     |
| ------------------ | ---- | ------------------------- | ---------------------- | ------- | --------- |
| `largefile.tar.gz` | 10GB | `[====>         ]` 35%    | `[======>       ]` 50% | `2h45m` | `3.6MB/s` |

### Custom Headers

Use custom HTTP headers for authentication or API-specific requirements:

```bash
gograb --header Authorization:BearerToken --header Accept:application/json https://api.example.com/securefile
```

Example headers:

| Header Key      | Value              |
| --------------- | ------------------ |
| `Authorization` | `BearerToken`      |
| `Accept`        | `application/json` |

### Proxy Support

Configure your HTTP or HTTPS proxy using environment variables:

```bash
export HTTP_PROXY=http://proxy.example.com:8080
export HTTPS_PROXY=https://proxy.example.com:8080

gograb https://example.com/file.zip
```

Example with proxy enabled:

| Proxy Configured  | File Name  | Download Speed | ETA      |
| ----------------- | ---------- | -------------- | -------- |
| `http://proxy...` | `file.zip` | `1.5MB/s`      | `10m30s` |

Why Use gograb?

gograb was built to address the unique challenges of downloading large files for modern workflows:

Use Case: Machine Learning Workflows

Machine learning practitioners often need to download pre-trained models, large datasets, or archives from unreliable sources. For example:

gograb https://storage.googleapis.com/models/bert-large.tar.gz https://datasets.example.com/imagenet.zip

## Performance

| Tool     | Speed (Multiple Downloads) | Speed (Single Download) | Resumable Downloads | Custom Headers |
| -------- | -------------------------- | ----------------------- | ------------------- | -------------- |
| `wget`   | Moderate                   | High                    | Partial Support     | Limited        |
| `curl`   | Moderate                   | High                    | Manual Setup        | Good           |
| **`gograb`** | **High**                   | **High**                | **Seamless**        | **Excellent**  |
