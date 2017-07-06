# httpstat

WIP Go HTTP tracing pkg. Not yet in production for [Apex Ping](https://apex.sh/ping/), but likely will be at some point.

## Example

```
res, _ := httpstat.Request("GET", "http://apex.sh", nil, nil)
enc := json.NewEncoder(os.Stderr)
enc.SetIndent("", "  ")
enc.Encode(res.Stats())
```

```
{
  "status": 200,
  "redirects": 1,
  "tls": true,
  "header": {
    "Age": [
      "2687"
    ],
    "Content-Length": [
      "9095"
    ],
    "Content-Type": [
      "text/html"
    ],
    "Date": [
      "Wed, 28 Jun 2017 03:13:39 GMT"
    ],
    "Etag": [
      "\"3889de5294fd56ac2596668b3943b58e\""
    ],
    "Last-Modified": [
      "Tue, 27 Jun 2017 18:32:26 GMT"
    ],
    "Server": [
      "AmazonS3"
    ],
    "Vary": [
      "Accept-Encoding"
    ],
    "Via": [
      "1.1 6a1e4dd9fa29c61c4b71a53d6bf94267.cloudfront.net (CloudFront)"
    ],
    "X-Amz-Cf-Id": [
      "zFqohdPbqpqeXvMdB2mr8m9JiUIBCw_tMfDUY_RfaOzvdvShxEub-Q=="
    ],
    "X-Cache": [
      "Hit from cloudfront"
    ]
  },
  "header_size": 396,
  "body_size": 9095,
  "time_dns": 1244723,
  "time_connect": 19610427,
  "time_tls": 101961103,
  "time_wait": 21167169,
  "time_response": 21330931,
  "time_download": 163762,
  "time_total": 145177108,
  "time_total_with_redirects": 193749062,
  "time_redirects": 48571954,
  "traces": [
    {
      "tls": false,
      "time_dns": 1789711,
      "time_connect": 21597753,
      "time_tls": 0,
      "time_wait": 24391045,
      "time_response": 169912459,
      "time_download": 145521414,
      "time_total": 193749608
    },
    {
      "tls": true,
      "time_dns": 1244723,
      "time_connect": 19610427,
      "time_tls": 101961103,
      "time_wait": 21167169,
      "time_response": 21338000,
      "time_download": 170831,
      "time_total": 145184177
    }
  ]
}
```

---

[![GoDoc](https://godoc.org/github.com/apex/httpstat?status.svg)](https://godoc.org/github.com/apex/httpstat)
![](https://img.shields.io/badge/license-MIT-blue.svg)
![](https://img.shields.io/badge/status-stable-green.svg)

<a href="https://apex.sh"><img src="http://tjholowaychuk.com:6000/svg/sponsor"></a>
