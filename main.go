package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/AndrewBlackwell/gograb/termutil"
	"github.com/urfave/cli"
)

// displayUsage provides the usage instructions for the program.
func displayUsage() {
	usage := `To use: grab [--header <header> [--header <header>]] [[rate limit:]url...]
--header: Specify your HTTP header in the format "key:value"
rate limit: limits the download speed, unit is in KBs
url...: URLs to download`
	fmt.Println(usage)
}

func main() {
	app := cli.NewApp()
	app.Name = "gograb"
	app.Flags = []cli.Flag{
		cli.StringSliceFlag{
			Name: "header",
		},
	}

	// Override the default help printer with our custom usage display.
	cli.HelpPrinter = func(w io.Writer, templ string, data interface{}) {
		displayUsage()
	}

	// Define the action executed when the program runs.
	app.Action = func(c *cli.Context) error {
		if c.NArg() == 0 {
			displayUsage()
			return nil
		}

		headers := c.StringSlice("header")
		headerMap := parseHeaders(headers)
		tasks := make([]*downloadTask, c.NArg())

		for i, url := range c.Args() {
			task := newDownloadTask(url, headerMap)
			if task != nil {
				go task.start()
				tasks[i] = task
			}
		}

		width, err := termutil.TerminalWidth()
		hasWidth := err == nil

		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		isFirstUpdate := true

		// Goroutine to update terminal output periodically.
		go func() {
			for {
				select {
				case <-ticker.C:
					if !isFirstUpdate {
						termutil.ClearLines(int16(len(tasks)))
					}
					updateTerminal(hasWidth, tasks, width)
					isFirstUpdate = false
				}
			}
		}()

		// Wait for all tasks to finish.
		for _, task := range tasks {
			if task != nil {
				<-task.completionChan
			}
		}

		time.Sleep(time.Second)
		fmt.Println("Download completed.")
		return nil
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

// updateTerminal refreshes the terminal output to show download progress.
func updateTerminal(hasWidth bool, tasks []*downloadTask, terminalWidth int) {
	for _, task := range tasks {
		var output string

		// Handle errors
		if task.error != nil && task.error != io.EOF {
			if task.fileName == "" {
				output = fmt.Sprintf("Error: %s", task.error.Error())
			} else {
				output = fmt.Sprintf("%s: Error: %s", task.fileName, task.error.Error())
			}
		} else if task.getBytesRead() > 0 {
			var etaInfo, fileSizeInfo, fileNameInfo string

			displayFileNameLength := 20
			fileNameInfo = truncateFileName(task.fileName, displayFileNameLength)

			if task.totalFileSize <= 0 {
				fileSizeInfo = fmt.Sprintf("|%s", humanReadableSize(task.getBytesRead()))
			} else {
				fileSizeInfo = fmt.Sprintf("|%s", humanReadableSize(task.totalFileSize))
			}

			etaInfo = fmt.Sprintf("%s|%s/s", task.getETAString(), task.getSpeedString())

			if hasWidth && task.totalFileSize > 0 {
				progressBarLength := terminalWidth - visibleWidth(fileSizeInfo+etaInfo) - displayFileNameLength
				if progressBarLength > 4 {
					fileSizeInfo += "["
					etaInfo = "]" + etaInfo

					ratio := float64(task.getBytesRead()) / float64(task.totalFileSize)
					progressBarLength -= 2
					bar := strings.Repeat(" ", progressBarLength)
					progressWidth := int(float64(progressBarLength) * ratio)
					progress := ""
					if progressWidth > 0 {
						progress = strings.Repeat("=", progressWidth)
					}
					if progressWidth+1 < len(bar) {
						bar = strings.Join([]string{progress, ">", bar[progressWidth+1:]}, "")
					} else {
						bar = strings.Join([]string{progress, ">"}, "")
					}
					output = strings.Join([]string{fileNameInfo, fileSizeInfo, bar, etaInfo}, "")
				} else if progressBarLength < 0 {
					output = output[:terminalWidth]
				} else {
					output = strings.Join([]string{fileNameInfo, fileSizeInfo, etaInfo}, "")
				}
			} else if task.totalFileSize > 0 {
				output = strings.Join([]string{fileNameInfo, fileSizeInfo, fmt.Sprintf("|%.2f%%", 100*float64(task.getBytesRead())/float64(task.totalFileSize)), etaInfo}, "")
			} else {
				output = strings.Join([]string{fileNameInfo, fmt.Sprintf("|%s", humanReadableSize(task.getBytesRead()))}, "")
			}
		} else {
			output = "Waiting..."
		}

		if hasWidth {
			outputWidth := visibleWidth(output)
			if outputWidth > terminalWidth {
				output = output[:terminalWidth]
			} else if outputWidth < terminalWidth {
				output += strings.Repeat(" ", terminalWidth-outputWidth)
			}
		}

		fmt.Println(output)
	}
}

// truncateFileName shortens or pads the filename to fit within a specific width.
func truncateFileName(fileName string, maxWidth int) string {
	if len(fileName) < maxWidth {
		return strings.Join([]string{fileName, strings.Repeat(" ", maxWidth-len(fileName))}, "")
	}

	runes := []rune(fileName)
	if len(runes) != len(fileName) {
		for {
			display := string(runes[:len(runes)])
			if visibleWidth(display) <= maxWidth {
				return strings.Join([]string{display, strings.Repeat(" ", maxWidth-visibleWidth(display))}, "")
			}
			runes = runes[:len(runes)-1]
		}
	}
	return fileName[:maxWidth]
}
