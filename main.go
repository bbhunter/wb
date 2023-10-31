package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"

	"log"
	"time"

	"io"
	"net/http"
	"os"
)

const (
	version = "0.0.10"

	wbSnapshotApiURL = "https://web.archive.org/cdx/search/xd?output=json&url=%s&fl=timestamp,original&collapse=digest&gzip=false&filter=statuscode:200"
	wbFileURL        = "https://web.archive.org/web/%sid_/%s"

	parseTimeLayout = "20060102150405"
	viewTimeLayout  = "2006-01-02 15:04:05"
)

var (
	flagUrl             = flag.String("u", "", "specify url")
	flagTimeout         = flag.Duration("t", 5*time.Second, "specify timeout")
	flagSnapshots       = flag.Bool("snapshots", false, "get all snapshots")
	flagDate            = flag.String("date", "", "get snapshot for a specific date")
	flagGetAllSnapshots = flag.Bool("all", false, "get all snapshots")
	flagHelp            = flag.Bool("help", false, "show help")
	flagNoBanner        = flag.Bool("no-banner", false, "hide banner")
)

func main() {
	flag.Parse()

	if !*flagNoBanner {
		fmt.Printf("🪄 wb / v%s\n----\n", version)
	}

	if *flagHelp {
		fmt.Println("Usage: \n	wb [flags]\n")
		flag.PrintDefaults()
		os.Exit(0)
	}

	var urls []string
	if *flagUrl != "" {
		urls = []string{*flagUrl}
	} else {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			urls = append(urls, scanner.Text())
		}
	}

	client := http.Client{
		Timeout: *flagTimeout,
		Transport: &http.Transport{
			TLSHandshakeTimeout: 10 * time.Second,
		},
	}

	for _, url := range urls {
		if len(urls) > 1 {
			fmt.Println("// Snapshots for", url)
		}
		snapshots, err := getSnapshots(client, url)
		if err != nil {
			log.Printf("failed to snapshots: %s\n", err)
			continue
		}

		if *flagSnapshots {
			fmt.Println("Snapshots for", url)
			for _, s := range snapshots {
				parsedTime, err := time.Parse(parseTimeLayout, s[0])
				if err != nil {
					log.Fatalf("failed to parse time: %s\n", err)
				}
				fmt.Printf("* %s | %s | %s\n", s[0], parsedTime.Format(viewTimeLayout), s[1])
			}
			continue
		}

		selectedSnapshot := snapshots[len(snapshots)-1]
		if *flagDate != "" {
			for _, s := range snapshots {
				if s[0] == *flagDate {
					selectedSnapshot = s
					break
				}
			}
		}

		if *flagGetAllSnapshots {
			for _, s := range snapshots {
				snapshotContent, err := getSnapshotContent(client, s[0], s[1])
				if err != nil {
					log.Printf("failed to read input: %s\n", err)
				}

				io.Copy(os.Stdout, snapshotContent)
			}
			continue
		}

		snapshotContent, err := getSnapshotContent(client, selectedSnapshot[0], selectedSnapshot[1])
		if err != nil {
			log.Printf("failed to read input: %s\n", err)
			continue
		}

		io.Copy(os.Stdout, snapshotContent)
	}
}

func getSnapshots(c http.Client, url string) ([][]string, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf(wbSnapshotApiURL, url), nil)
	if err != nil {
		return [][]string{}, fmt.Errorf("getSnapshots: failed to generate request waybackmachine api: %w url: %s", err, url)
	}

	rsp, err := c.Do(req)
	if err != nil {
		return [][]string{}, fmt.Errorf("getSnapshots: failed to send request waybackmachine api: %w url: %s", err, url)
	}
	defer rsp.Body.Close()

	var r [][]string
	dec := json.NewDecoder(rsp.Body)

	err = dec.Decode(&r)
	if err != nil {
		return [][]string{}, fmt.Errorf("getSnapshots: error while decoding response %w url: %s", err, url)
	}

	if len(r) < 1 {
		return [][]string{}, fmt.Errorf("getSnapshots: no results found for this url: %s", url)
	}

	return r[1:], nil
}

func getSnapshotContent(c http.Client, ts, url string) (io.ReadCloser, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf(wbFileURL, ts, url), nil)
	if err != nil {
		return nil, fmt.Errorf("getSnapshotContent: failed to generate request waybackmachine api: %w url: %s", err, url)
	}
	req.Header.Add("Accept-Encoding", "plain")

	rsp, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("getSnapshotContent: failed to send request waybackmachine api: %w url: %s", err, url)
	}

	return rsp.Body, nil
}
