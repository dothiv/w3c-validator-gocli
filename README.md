# CLI for the W3C validtor

Command line interface for recursively validating websites with the W3C validator

This could be handy if you are testing private webpages, or want to integrate this into your CI.

## How to use

Have a W3C validator instance ready, e.g. by using this docker file: [dockerhtml5validator](https://github.com/magnetikonline/dockerhtml5validator).

### Install

    go get github.com/dothiv/w3c-validator-gocli/w3c-validator-gocli
    
### Use

    $GOPATH/bin/w3c-validator-gocli -validator="http://localhost:8080/check" -url="http://google.com/"
