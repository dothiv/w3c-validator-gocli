package linkextractor

import (
	"fmt"
	neturl "net/url"
	"os"
	"regexp"
)

var HYPERLINK = regexp.MustCompile(`<a[^>]+href="([^"]+)"`)

func GetLinks(source []byte, url *neturl.URL) (links []neturl.URL) {
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
			linkUrl.Fragment = "" // Do not follow fragments
			links = append(links, *linkUrl)
		}
	}
	return
}
