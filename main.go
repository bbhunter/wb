package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	log "github.com/sirupsen/logrus"
	"io"
	"net/http"
	"os"
	"time"
)

const (
	version = "0.0.4"

	wbSnapshotApiURL = "https://web.archive.org/cdx/search/xd?output=json&url=%s&fl=timestamp,original&collapse=digest&gzip=false&filter=statuscode:200"
	wbFileURL        = "https://web.archive.org/web/%sid_/%s"

	parseTimeLayout = "20060102150405"
	viewTimeLayout  = "2006-01-02 15:04:05"
)

var (
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
		fmt.Println("Usage: \n	wb <url> [flags]\n")
		flag.PrintDefaults()
		os.Exit(0)
	}

	var url string
	if flag.NArg() > 0 {
		url = flag.Arg(0)
	} else {
		sc := bufio.NewScanner(os.Stdin)
		for sc.Scan() {
			url = sc.Text()
		}

		if err := sc.Err(); err != nil {
			log.Fatalf("failed to read input: %s\n", err)
		}
	}

	client := http.Client{
		Timeout: time.Second * 5,
	}

	snapshots, err := getSnapshots(client, url)
	if err != nil {
		log.Fatalf("failed to snapshots: %s\n", err)
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
		os.Exit(0)
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
				log.Fatalf("failed to read input: %s\n", err)
			}

			io.Copy(os.Stdout, snapshotContent)
		}
		os.Exit(0)
	}

	snapshotContent, err := getSnapshotContent(client, selectedSnapshot[0], selectedSnapshot[1])
	if err != nil {
		log.Fatalf("failed to read input: %s\n", err)
	}

	io.Copy(os.Stdout, snapshotContent)
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
