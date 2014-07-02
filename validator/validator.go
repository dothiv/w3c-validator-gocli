package validator

import (
	"bytes"
	"fmt"
	"github.com/dothiv/w3c-validator-gocli/linkextractor"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/textproto"
	neturl "net/url"
	"os"
	"strings"
)

type Validator struct {
	checkStatusCode bool
	printMessage    bool
	recursive       bool
	checkedUrls     map[string]bool
	validator       *neturl.URL
}

func NewValidator(validator *neturl.URL) (v *Validator) {
	v = new(Validator)
	v.checkStatusCode = false
	v.printMessage = false
	v.recursive = true
	v.validator = validator
	v.checkedUrls = make(map[string]bool)
	return
}

func (v *Validator) CheckStatusCode(b bool) {
	v.checkStatusCode = b
}

func (v *Validator) PrintMessage(b bool) {
	v.printMessage = b
}

func (v *Validator) Recursive(b bool) {
	v.recursive = b
}

/**
 * Recursively check a page and linked sub pages of the same domain.
 */
func (v *Validator) RecursiveCheck(pageUrl *neturl.URL, startUrl *neturl.URL) {
	pageSource, err := v.checkUrl(pageUrl)
	if err != nil {
		os.Stderr.WriteString(fmt.Sprintf("[ERROR] %s\n", pageUrl))
		os.Stderr.WriteString(fmt.Sprintf("%s\n", err.Error()))
		v.checkedUrls[pageUrl.String()] = false
	} else {
		os.Stdout.WriteString(fmt.Sprintf("[OK] %s\n", pageUrl))
		v.checkedUrls[pageUrl.String()] = true
	}
	if !v.recursive {
		return
	}
	links := linkextractor.GetLinks(pageSource, startUrl)
	for _, link := range links {
		if _, checked := v.checkedUrls[link.String()]; !checked {
			v.RecursiveCheck(&link, startUrl)
		}
	}
}

/**
 * Checks the validity of a document by fetching it and sending to the validator instance.
 */
func (v *Validator) checkUrl(url *neturl.URL) (fetchContents []byte, checkErr error) {
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
	if v.checkStatusCode && headResponse.StatusCode != 200 {
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
	postResponse, checkErr = http.Post(v.validator.String(), "multipart/form-data", &b)
	if checkErr != nil {
		return
	}

	status := postResponse.Header.Get("X-W3C-Validator-Status")
	if status != "Valid" {
		checkErr = fmt.Errorf("%s!", status)
		if v.printMessage {
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
