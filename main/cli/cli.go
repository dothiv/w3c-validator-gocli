//
// Command line interface for recursively validating websites with the W3C validator
//
// Copyright 2014 TLD dotHIV Registry GmbH.
// @author Markus Tacker <m@dotHIV.org>
//
package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/textproto"
	neturl "net/url"
	"os"
	"regexp"
	"strings"
)

func main() {
	url := flag.String("url", "", "URL to start validation of")
	validator := flag.String("validator", "http://localhost:8080/check", "W3C validation service")
	ignore_status := flag.String("ignore-status", "0", "Accept status codes other than 200")
	print_message := flag.String("print-message", "0", "Print validation message")
	flag.Parse()

	if len(*url) == 0 {
		os.Stderr.WriteString("url is required\n")
		flag.Usage()
		os.Exit(1)
	}

	if len(*validator) == 0 {
		os.Stderr.WriteString("validator service is required\n")
		flag.Usage()
		os.Exit(1)
	}

	pageUrl, pageUrlErr := neturl.Parse(*url)
	if pageUrlErr != nil {
		os.Stderr.WriteString(pageUrlErr.Error())
		os.Exit(1)
	}

	validatorUrl, validatorUrlErr := neturl.Parse(*validator)
	if validatorUrlErr != nil {
		os.Stderr.WriteString(validatorUrlErr.Error())
		os.Exit(1)
	}

	os.Stdout.WriteString(fmt.Sprintf("Using %s ...\n", *validator))

	checkedUrls := make(map[string]bool)

	checkStatusCode := true
	if *ignore_status != "0" {
		checkStatusCode = false
	}

	printMessage := false
	if *print_message != "0" {
		printMessage = true
	}

	recursiveCheck(pageUrl, pageUrl, validatorUrl, checkedUrls, checkStatusCode, printMessage)
	return
}

/**
 * Recursively check a page and linked sub pages of the same domain.
 */
func recursiveCheck(pageUrl *neturl.URL, startUrl *neturl.URL, validator *neturl.URL, checkedUrls map[string]bool, checkStatusCode bool, printMessage bool) {
	pageSource, err := checkUrl(pageUrl, validator, checkStatusCode, printMessage)
	if err != nil {
		os.Stderr.WriteString(fmt.Sprintf("[ERROR] %s\n", pageUrl))
		os.Stderr.WriteString(fmt.Sprintf("%s\n", err.Error()))
		checkedUrls[pageUrl.String()] = false
	} else {
		os.Stdout.WriteString(fmt.Sprintf("[OK] %s\n", pageUrl))
		checkedUrls[pageUrl.String()] = true
	}
	links := getLinks(pageSource, startUrl)
	for _, link := range links {
		if _, checked := checkedUrls[link.String()]; !checked {
			recursiveCheck(&link, startUrl, validator, checkedUrls, checkStatusCode, printMessage)
		}
	}
}

/**
 * Checks the validity of a document by fetching it and sending to the validator instance.
 */
func checkUrl(url *neturl.URL, validator *neturl.URL, checkStatusCode bool, printMessage bool) (fetchContents []byte, checkErr error) {
	// Check URLs content type
	var headResponse *http.Response
	headResponse, checkErr = http.Head(url.String())
	if checkErr != nil {
		return
	}
	contentType := headResponse.Header.Get("content-type")
	if !strings.Contains(contentType, "text/html") {
		checkErr = fmt.Errorf("%s not supported", contentType)
		return
	}
	if checkStatusCode && headResponse.StatusCode != 200 {
		checkErr = fmt.Errorf("Status %d!", headResponse.StatusCode)
		return
	}

	// Fetch url
	var tempFile *os.File
	tempFile, checkErr = ioutil.TempFile(os.TempDir(), "prefix")
	defer os.Remove(tempFile.Name())
	if checkErr != nil {
		return
	}

	var fetchResponse *http.Response
	fetchResponse, checkErr = http.Get(url.String())
	if checkErr != nil {
		return
	}
	defer fetchResponse.Body.Close()

	fetchContents, checkErr = ioutil.ReadAll(fetchResponse.Body)
	if checkErr != nil {
		return
	}
	tempFile.Write(fetchContents)
	tempFile.Seek(0, 0)

	// Build request
	var quoteEscaper = strings.NewReplacer("\\", "\\\\", `"`, "\\\"")

	var b bytes.Buffer
	w := multipart.NewWriter(&b)

	// Add page
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="uploaded_file"; filename="%s"`, quoteEscaper.Replace(url.String())))
	h.Set("Content-Type", fetchResponse.Header.Get("content-type"))

	var fw io.Writer
	fw, checkErr = w.CreatePart(h)
	if checkErr != nil {
		return
	}
	_, checkErr = io.Copy(fw, tempFile)
	if checkErr != nil {
		return
	}

	// Add output type
	var outputField io.Writer
	outputField, checkErr = w.CreateFormField("output")
	if checkErr != nil {
		return
	}
	outputField.Write([]byte("soap12"))

	// Done.
	w.Close()

	// Talk to service
	var postResponse *http.Response
	postResponse, checkErr = http.Post(validator.String(), "multipart/form-data", &b)
	if checkErr != nil {
		return
	}

	status := postResponse.Header.Get("X-W3C-Validator-Status")
	if status != "Valid" {
		checkErr = fmt.Errorf("%s!", status)
		if printMessage {
			validatorContents, responseReadErr := ioutil.ReadAll(postResponse.Body)
			if responseReadErr != nil {
				return
			}
			os.Stderr.Write(validatorContents)
			os.Stderr.WriteString("\n")
		}
		return
	}
	return
}

var HYPERLINK = regexp.MustCompile(`<a[^>]+href="([^"]+)"`)

func getLinks(source []byte, url *neturl.URL) (links []neturl.URL) {
	all := HYPERLINK.FindAllSubmatch(source, -1)
	for _, l := range all {
		if l[1][0] != '/' {
			continue
		}
		if len(l[1]) >= 2 && l[1][1] == '/' { // double slash prefixes -> different host
			continue
		}
		linkUrl, linkUrlErr := neturl.Parse(fmt.Sprintf("%s://%s%s", url.Scheme, url.Host, l[1]))
		if linkUrlErr != nil {
			os.Stderr.WriteString(linkUrlErr.Error())
		} else {
			links = append(links, *linkUrl)
		}
	}
	return
}
