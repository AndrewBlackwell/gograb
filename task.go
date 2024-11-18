package main

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

type downloadTask struct {
	completionChan chan struct{}
	source         io.ReadCloser
	destination    io.WriteCloser
	bytesPerSecond float64
	error          error
	startTime      time.Time
	endTime        time.Time
	mutex          sync.Mutex
	bytesRead      int64
	totalFileSize  int64
	fileName       string
	buffer         []byte
	rateLimiter    *rateLimiter
	downloadURL    string
	isResumable    bool
	headers        map[string]string
}

// getBytesRead returns the number of bytes read so far.
func (dt *downloadTask) getBytesRead() int64 {
	if dt == nil {
		return 0
	}
	return atomic.LoadInt64(&dt.bytesRead)
}

// newDownloadTask initializes a new download task.
func newDownloadTask(url string, headers map[string]string) *downloadTask {
	limit, url := extractRateLimit(url)
	return &downloadTask{
		downloadURL:    url,
		completionChan: make(chan struct{}, 1),
		buffer:         make([]byte, 32*1024),
		rateLimiter:    &rateLimiter{limit: limit * 1000},
		headers:        headers,
	}
}

// start begins the download task.
func (dt *downloadTask) start() {
	defer func() {
		if err := recover(); err != nil {
			switch e := err.(type) {
			case string:
				dt.error = errors.New(e)
			case error:
				dt.error = e
			default:
				dt.error = errors.New("unknown panic occurred")
			}
			close(dt.completionChan)
			dt.endTime = time.Now()
		}
	}()

	var destinationFile *os.File
	var bytesRead, bytesWritten int
	var fileName string
	var fileInfo os.FileInfo

	// Create HTTP request
	request, _ := http.NewRequest("GET", dt.downloadURL, nil)
	if dt.headers != nil {
		for key, value := range dt.headers {
			request.Header.Set(key, value)
		}
	}

	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
		},
	}
	response, err := client.Do(request)
	if err != nil || (response.StatusCode != http.StatusOK && response.StatusCode != http.StatusPartialContent) {
		dt.error = fmt.Errorf("HTTP request failed with status: %d", response.StatusCode)
		close(dt.completionChan)
		dt.endTime = time.Now()
		return
	}

	fileName, err = extractFilename(response)

	fileInfo, err = os.Stat(fileName)
	if err == nil {
		if !fileInfo.IsDir() {
			response.Body.Close()
			if fileInfo.Size() == response.ContentLength {
				dt.error = errors.New("file already downloaded")
				close(dt.completionChan)
				dt.endTime = time.Now()
				return
			}
			request.Header.Set("Range", fmt.Sprintf("bytes=%d-", fileInfo.Size()))
			response, err = client.Do(request)
			if err != nil || (response.StatusCode != http.StatusOK && response.StatusCode != http.StatusPartialContent) {
				dt.error = fmt.Errorf("HTTP request failed with status: %d", response.StatusCode)
				close(dt.completionChan)
				dt.endTime = time.Now()
				return
			}
			if response.Header.Get("Accept-Ranges") == "bytes" || response.Header.Get("Content-Range") != "" {
				destinationFile, err = os.OpenFile(fileName, os.O_RDWR, 0666)
				if err != nil {
					close(dt.completionChan)
					dt.endTime = time.Now()
					return
				}
				destinationFile.Seek(0, os.SEEK_END)
				dt.bytesRead = fileInfo.Size()
				dt.isResumable = true
			}
		}
	}

	if destinationFile == nil {
		destinationFile, err = os.Create(fileName)
		if err != nil {
			close(dt.completionChan)
			dt.endTime = time.Now()
			return
		}
	}

	dt.destination = destinationFile
	dt.source = response.Body
	dt.fileName = fileName
	if response.ContentLength > 0 && dt.isResumable && fileInfo != nil {
		dt.totalFileSize = response.ContentLength + fileInfo.Size()
	} else {
		dt.totalFileSize = response.ContentLength
	}

	go dt.monitorSpeed()

	dt.startTime = time.Now()

	for {
		if dt.rateLimiter.limit > 0 {
			dt.rateLimiter.wait(dt.bytesRead)
		}

		bytesRead, err = dt.source.Read(dt.buffer)
		if bytesRead > 0 {
			bytesWritten, err = dt.destination.Write(dt.buffer[:bytesRead])
			if err != nil || bytesRead != bytesWritten {
				dt.error = io.ErrShortWrite
				break
			}
			atomic.AddInt64(&dt.bytesRead, int64(bytesRead))
		}

		if err != nil {
			break
		}
	}

	dt.error = err
	close(dt.completionChan)
	dt.endTime = time.Now()
}

// monitorSpeed calculates the download speed periodically.
func (dt *downloadTask) monitorSpeed() {
	var previousBytes int64
	lastCheck := dt.startTime

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-dt.completionChan:
			return
		case now := <-ticker.C:
			duration := now.Sub(lastCheck)
			lastCheck = now

			currentBytes := dt.getBytesRead()
			bytesDownloaded := currentBytes - previousBytes
			previousBytes = currentBytes

			dt.mutex.Lock()
			dt.bytesPerSecond = float64(bytesDownloaded) / duration.Seconds()
			dt.mutex.Unlock()
		}
	}
}

// getSpeedString returns the current download speed as a human-readable string.
func (dt *downloadTask) getSpeedString() string {
	dt.mutex.Lock()
	defer dt.mutex.Unlock()
	return humanReadableSize(int64(dt.bytesPerSecond))
}

// getETAString calculates and returns the estimated time remaining as a string.
func (dt *downloadTask) getETAString() string {
	dt.mutex.Lock()
	defer dt.mutex.Unlock()
	if dt.totalFileSize == 0 || dt.bytesPerSecond == 0 {
		return "N/A"
	}
	remainingTime := (dt.totalFileSize - dt.getBytesRead()) / int64(dt.bytesPerSecond)
	return durationToString(remainingTime)
}
