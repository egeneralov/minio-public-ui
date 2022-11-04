package main

import (
  "context"
  "crypto/tls"
  "encoding/xml"
  "flag"
  "fmt"
  "io/ioutil"
  "net"
  "net/http"
  "time"
)

//-url "https://s3-a4.generalov.org/android?max-keys=1000" -eTag -lastModified -ownerDisplayName -size -storageClass
var (
  lastModified     = flag.Bool("lastModified", false, "show lastModified")
  eTag             = flag.Bool("eTag", false, "show eTag")
  size             = flag.Bool("size", false, "show size")
  ownerDisplayName = flag.Bool("ownerDisplayName", false, "show ownerDisplayName")
  storageClass     = flag.Bool("storageClass", false, "show storageClass")
  insecure         = flag.Bool("insecure", false, "skip TLS verification")
  url              = flag.String("url", "http://localhost:9000/minio-public-ui/?&max-keys=1000", "url")
  title            = flag.String("title", "Minio Public UI", "title")
  bind             = flag.String("bind", "0.0.0.0:8080", "bind")
  customCSS        = flag.String("custom-css-file", "", "path to custom css file")
)

var heading = `
<style>
table {
  font-family: helvetica;
  border-collapse: collapse;
  width: 100%;
  align: center;
  text-align: center;
  vertical-align: bottom;
  border: 1px solid #ddd;
}
</style>
`

func init() {
  flag.Parse()
  if *customCSS == "" {
    return
  }
  css, err := ioutil.ReadFile(*customCSS)
  if err != nil {
    panic(err)
  }
  heading = fmt.Sprintf("<style>%s</style>", css)
}

func main() {
  http.HandleFunc("/", handler)
  http.ListenAndServe(*bind, nil)
}

func handler(w http.ResponseWriter, req *http.Request) {
  w.Header().Set("Content-Type", "text/html")
  w.Write([]byte(render(*title, *url)))
}

func render(title, url string) string {
  page := fmt.Sprintf("<html><head><title>%s</title>%s</head>", title, heading)
  r, err := getListBucketResult(url)
  if err != nil {
    page += renderBodyWithError(err)
  } else {
    page += renderBodyWithResult(r)
  }
  page += "</html>"
  return page
}

func renderBodyWithError(err error) string {
  return "<body>" + err.Error() + "</body>"
}

func renderBodyWithResult(r ListBucketResult) string {
  result := "<body><table><thead><th>Key</th>"
  if *lastModified {
    result += "<th>LastModified</th>"
  }
  if *eTag {
    result += "<th>ETag</th>"
  }
  if *size {
    result += "<th>Size</th>"
  }
  if *ownerDisplayName {
    result += "<th>Owner</th>"
  }
  if *storageClass {
    result += "<th>StorageClass</th>"
  }
  result += "</thead><tbody>"
  for _, el := range r.Contents {
    var element string
    element = fmt.Sprintf("\n<tr>\n\t<td><a href=\"%s\">%s</a></td>", el.Key, el.Key)
    if *lastModified {
      element += fmt.Sprintf("\n\t<td>%s</td>", el.LastModified.Format(time.RFC3339))
    }
    if *eTag {
      element += fmt.Sprintf("\n\t<td>%s</td>", el.ETag)
    }
    if *size {
      element += fmt.Sprintf("\n\t<td>%d</td>", el.Size)
    }
    if *ownerDisplayName {
      element += fmt.Sprintf("\n\t<td>%s</td>", el.Owner.DisplayName)
    }
    if *storageClass {
      element += fmt.Sprintf("\n\t<td>%s</td>", el.StorageClass)
    }
    element += "</tr>\n"
    result += element
  }
  result += "</tbody></table></body>"
  return result
}

func defaultTransportDialContext(dialer *net.Dialer) func(context.Context, string, string) (net.Conn, error) {
  return dialer.DialContext
}

func getListBucketResult(url string) (ListBucketResult, error) {
  var (
    result     ListBucketResult
    httpClient = &http.Client{
      Transport: &http.Transport{
        Proxy: http.ProxyFromEnvironment,
        DialContext: defaultTransportDialContext(&net.Dialer{
          Timeout:   30 * time.Second,
          KeepAlive: 30 * time.Second,
        }),
        ForceAttemptHTTP2:     true,
        MaxIdleConns:          100,
        IdleConnTimeout:       90 * time.Second,
        TLSHandshakeTimeout:   10 * time.Second,
        ExpectContinueTimeout: 1 * time.Second,
        TLSClientConfig: &tls.Config{
          InsecureSkipVerify: *insecure,
        },
      },
      Timeout: 10 * time.Second,
    }
  )
  resp, err := httpClient.Get(url)
  if err != nil {
    return result, err
  }
  defer resp.Body.Close()
  body, err := ioutil.ReadAll(resp.Body)
  if err != nil {
    return result, err
  }
  err = xml.Unmarshal(body, &result)
  if err != nil {
    return result, err
  }
  return result, nil
}

type ListBucketResult struct {
  XMLName     xml.Name `xml:"ListBucketResult"`
  Text        string   `xml:",chardata"`
  Xmlns       string   `xml:"xmlns,attr"`
  Name        string   `xml:"Name"`
  Prefix      string   `xml:"Prefix"`
  Marker      string   `xml:"Marker"`
  MaxKeys     string   `xml:"MaxKeys"`
  Delimiter   string   `xml:"Delimiter"`
  IsTruncated string   `xml:"IsTruncated"`
  Contents    []struct {
    Text         string    `xml:",chardata"`
    Key          string    `xml:"Key"`
    LastModified time.Time `xml:"LastModified"`
    ETag         string    `xml:"ETag"`
    Size         int64     `xml:"Size"`
    Owner        struct {
      Text        string `xml:",chardata"`
      ID          string `xml:"ID"`
      DisplayName string `xml:"DisplayName"`
    } `xml:"Owner"`
    StorageClass string `xml:"StorageClass"`
  } `xml:"Contents"`
}
