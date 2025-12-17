package main

import (
	"encoding/xml"
	"flag"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"time"
)

type Assertions struct {
	XMLName xml.Name `xml:"assertions"`
	Asserts []Assert `xml:"assert"`
}

type Assert struct {
	ID   string `xml:"id,attr"`
	Test Test   `xml:"test"`
}

type Test struct {
	ID          string  `xml:"id,attr"`
	Conformance string  `xml:"conformance,attr"`
	Manual      string  `xml:"manual,attr"`
	Starts      []Start `xml:"start"`
	Deps        []Dep   `xml:"dep"`
}

type Start struct {
	URI string `xml:"uri,attr"`
}

type Dep struct {
	URI string `xml:"uri,attr"`
}

const (
	BASE_URL      = "https://www.w3.org/Voice/2013/scxml-irp/"
	MANIFEST_URL  = BASE_URL + "manifest.xml"
	TEST_BASE_URL = BASE_URL
)

func downloadWithBackoff(url, localPath string) error {
	maxRetries := 5
	baseDelay := 1 * time.Second

	for attempt := 0; attempt <= maxRetries; attempt++ {
		resp, err := http.Get(url)
		if err != nil {
			if attempt == maxRetries {
				return fmt.Errorf("fetch error after %d retries: %w", maxRetries, err)
			}
			delay := time.Duration(math.Pow(2, float64(attempt))) * baseDelay
			time.Sleep(delay)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			out, err := os.Create(localPath)
			if err != nil {
				return fmt.Errorf("create file: %w", err)
			}
			defer out.Close()

			_, err = io.Copy(out, resp.Body)
			if err != nil {
				return fmt.Errorf("save file: %w", err)
			}
			return nil
		}

		if attempt == maxRetries {
			return fmt.Errorf("HTTP %d after %d retries", resp.StatusCode, maxRetries)
		}

		delay := time.Duration(math.Pow(2, float64(attempt))) * baseDelay
		time.Sleep(delay)
	}
	return fmt.Errorf("max retries exceeded")
}

func downloadManifest(force bool) error {
	localManifest := filepath.Join("pkg", "scxml_test_suite", "manifest.xml")
	if !force {
		if _, err := os.Stat(localManifest); err == nil {
			return nil
		}
	}
	if err := os.MkdirAll(filepath.Dir(localManifest), 0755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}
	return downloadWithBackoff(MANIFEST_URL, localManifest)
}

func getTestURIs(manifestPath string) ([]string, error) {
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("read manifest: %w", err)
	}
	var assertions Assertions
	if err := xml.Unmarshal(data, &assertions); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}
	uris := make(map[string]struct{})
	for _, assert := range assertions.Asserts {
		for _, start := range assert.Test.Starts {
			uris[start.URI] = struct{}{}
		}
		for _, dep := range assert.Test.Deps {
			uris[dep.URI] = struct{}{}
		}
	}
	var list []string
	for uri := range uris {
		list = append(list, uri)
	}
	sort.Strings(list)
	return list, nil
}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [-f] [--filepath DIR]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  -f             force download of manifest and tests (overwrite if exists)\n")
		fmt.Fprintf(os.Stderr, "  --filepath DIR directory to save downloaded test files (default \".\")\n")
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  %s\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -f\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s --filepath ./tests\n", os.Args[0])
	}
	forcePtr := flag.Bool("f", false, "force download (overwrite if exists)")
	filepathPtr := flag.String("filepath", ".", "directory to save downloaded test files")
	flag.Parse()

	args := flag.Args()
	if len(args) > 0 {
		flag.Usage()
		os.Exit(1)
	}

	// 1. Download/ensure manifest
	if err := downloadManifest(*forcePtr); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to ensure manifest: %v\n", err)
		os.Exit(1)
	}

	// 2. Parse manifest
	manifestPath := filepath.Join("pkg", "scxml_test_suite", "manifest.xml")
	uris, err := getTestURIs(manifestPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to parse manifest: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Found %d unique test URIs in manifest\n", len(uris))

	// 3. Download tests with checks
	skipped := 0
	downloaded := 0
	for _, relPath := range uris {
		fullURL := TEST_BASE_URL + relPath
		localPath := filepath.Join(*filepathPtr, relPath)

		// Check if exists
		if _, err := os.Stat(localPath); err == nil {
			if !*forcePtr {
				fmt.Printf("Skipping existing: %s\n", localPath)
				skipped++
				continue
			}
			fmt.Printf("Force overwriting: %s\n", localPath)
		} else if !os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "Error checking %s: %v\n", localPath, err)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating directory %s: %v\n", filepath.Dir(localPath), err)
			continue
		}

		if err := downloadWithBackoff(fullURL, localPath); err != nil {
			fmt.Fprintf(os.Stderr, "Download failed %s: %v\n", localPath, err)
			continue
		}

		fmt.Printf("Downloaded %s to %s\n", fullURL, localPath)
		downloaded++
	}
	fmt.Printf("Downloaded %d, skipped %d (total %d)\n", downloaded, skipped, len(uris))
}
