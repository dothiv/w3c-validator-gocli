//
// Command line interface for recursively validating websites with the W3C validator
//
// Copyright 2014 TLD dotHIV Registry GmbH.
// @author Markus Tacker <m@dotHIV.org>
//
package main

import (
	"flag"
	"fmt"
	"github.com/dothiv/w3c-validator-gocli/validator"
	neturl "net/url"
	"os"
)

func main() {
	url := flag.String("url", "", "URL to start validation of")
	validator_url := flag.String("validator", "http://localhost:8080/check", "W3C validation service")
	ignore_status := flag.String("ignore-status", "0", "Accept status codes other than 200")
	print_message := flag.String("print-message", "0", "Print validation message")
	no_follow := flag.String("no-follow", "0", "Do not follow links")
	flag.Parse()

	if len(*url) == 0 {
		os.Stderr.WriteString("url is required\n")
		flag.Usage()
		os.Exit(1)
	}

	if len(*validator_url) == 0 {
		os.Stderr.WriteString("validator service is required\n")
		flag.Usage()
		os.Exit(1)
	}

	pageUrl, pageUrlErr := neturl.Parse(*url)
	if pageUrlErr != nil {
		os.Stderr.WriteString(pageUrlErr.Error())
		os.Exit(1)
	}

	validatorUrl, validatorUrlErr := neturl.Parse(*validator_url)
	if validatorUrlErr != nil {
		os.Stderr.WriteString(validatorUrlErr.Error())
		os.Exit(1)
	}

	os.Stdout.WriteString(fmt.Sprintf("Using %s ...\n", *validator_url))

	v := validator.NewValidator(validatorUrl)

	if *ignore_status != "0" {
		v.CheckStatusCode(false)
	}

	if *print_message != "0" {
		v.PrintMessage(true)
	}

	if *no_follow == "1" {
		v.Recursive(false)
	}

	v.RecursiveCheck(pageUrl, pageUrl)
	return
}
