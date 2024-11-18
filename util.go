package main

import (
	"errors"
	"fmt"
	"mime"
	"net/http"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/mattn/go-runewidth.v0"
)

const (
	Kilobyte = 1024
	Megabyte = 1024 * Kilobyte
	Gigabyte = 1024 * Megabyte
	Terabyte = 1024 * Gigabyte
)

// humanReadableSize formats bytes into a human-readable string.
func humanReadableSize(size int64) string {
	switch {
	case size >= Terabyte:
		return fmt.Sprintf("%6.2fTB", float64(size)/Terabyte)
	case size >= Gigabyte:
		return fmt.Sprintf("%6.2fGB", float64(size)/Gigabyte)
	case size >= Megabyte:
		return fmt.Sprintf("%6.2fMB", float64(size)/Megabyte)
	case size >= Kilobyte:
		return fmt.Sprintf("%6.2fKB", float64(size)/Kilobyte)
	default:
		return fmt.Sprintf("%7dB", size)
	}
}

// durationToString converts a duration in seconds to a readable string.
func durationToString(seconds int64) string {
	switch {
	case seconds < 60:
		return fmt.Sprintf("%2ds", seconds)
	case seconds < 3600:
		minutes := seconds / 60
		remainder := seconds % 60
		if remainder == 0 {
			return fmt.Sprintf("%2dm", minutes)
		}
		return fmt.Sprintf("%2dm%2ds", minutes, remainder)
	default:
		hours := seconds / 3600
		remainder := seconds % 3600
		if remainder == 0 {
			return fmt.Sprintf("%2dh", hours)
		}
		return fmt.Sprintf("%2dh%s", hours, durationToString(remainder))
	}
}

var ErrMissingFilename = errors.New("unable to determine filename")

// extractFilename attempts to derive a filename from the HTTP response.
func extractFilename(response *http.Response) (string, error) {
	filename := response.Request.URL.Path
	if contentDisposition := response.Header.Get("Content-Disposition"); contentDisposition != "" {
		if _, params, err := mime.ParseMediaType(contentDisposition); err == nil {
			filename = params["filename"]
		}
	}

	if filename == "" || strings.HasSuffix(filename, "/") || strings.Contains(filename, "\x00") {
		return "", ErrMissingFilename
	}

	filename = filepath.Base(path.Clean("/" + filename))
	if filename == "" || filename == "." || filename == "/" {
		return "", ErrMissingFilename
	}

	return filename, nil
}

var ansiEscapeRegex = regexp.MustCompile("\x1b\x5b[0-9]+\x6d")

// visibleWidth calculates the visible width of a string by ignoring ANSI escape codes.
func visibleWidth(input string) int {
	width := runewidth.StringWidth(input)
	for _, match := range ansiEscapeRegex.FindAllString(input, -1) {
		width -= runewidth.StringWidth(match)
	}
	return width
}

// extractRateLimit splits a URL into a speed limit and the actual URL.
func extractRateLimit(url string) (int64, string) {
	parts := strings.Split(url, ":")
	if len(parts) >= 2 {
		limit, err := strconv.ParseInt(parts[0], 0, 0)
		if err != nil {
			return -1, url
		}
		return limit, strings.Join(parts[1:], ":")
	}
	return -1, url
}

// parseHeaders converts a slice of header strings into a map.
func parseHeaders(headerStrings []string) map[string]string {
	headers := make(map[string]string)
	for _, header := range headerStrings {
		if strings.Contains(header, ":") {
			parts := strings.SplitN(header, ":", 2)
			headers[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}
	return headers
}
