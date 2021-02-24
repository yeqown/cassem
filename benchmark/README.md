## Benchmark

Testing the core RESTful APIs basic `QPS` and try to optimize them.

### introduction to `go-wrk`

```sh
Usage of go-wrk:
  -CA string
    	A PEM eoncoded CA's certificate file. (default "someCertCAFile")
  -H string
    	the http headers sent separated by '\n' (default "User-Agent: go-wrk 0.1 benchmark\nContent-Type: text/html;")
  -b string
    	the http request body
  -c int
    	the max numbers of connections used (default 100)
  -cert string
    	A PEM eoncoded certificate file. (default "someCertFile")
  -d string
    	dist mode
  -f string
    	json config file
  -i	TLS checks are disabled
  -k	if keep-alives are disabled (default true)
  -key string
    	A PEM encoded private key file. (default "someKeyFile")
  -m string
    	the http request method (default "GET")
  -n int
    	the total number of calls processed (default 1000)
  -p string
    	the http request body data file
  -r	in the case of having stream or file in the response,
    	 it reads all response body to calculate the response size
  -s string
    	if specified, it counts how often the searched string s is contained in the responses
  -t int
    	the numbers of threads used (default 1)
```

### API (/api/namespaces) benchmark result

```sh
$ go-wrk -n=1000 -c=50 -t=10 http://localhost:2021/api/namespaces\?limit\=1\&offset\=0\&key\=ns
==========================BENCHMARK==========================
URL:				http://localhost:2021/api/namespaces?limit=1&offset=0&key=ns

Used Connections:		50
Used Threads:			10
Total number of calls:		1000

===========================TIMINGS===========================
Total time passed:		1.51s
Avg time per request:		71.48ms
Requests per second:		661.22
Median time per request:	64.60ms
99th percentile time:		255.75ms
Slowest time for request:	304.00ms

=============================DATA=============================
Total response body sizes:		46000
Avg response body per request:		46.00 Byte
Transfer rate per second:		30416.18 Byte/s (0.03 MByte/s)
==========================RESPONSES==========================
20X Responses:		1000	(100.00%)
30X Responses:		0	(0.00%)
40X Responses:		0	(0.00%)
50X Responses:		0	(0.00%)
Errors:			0	(0.00%)
```

### API (/api/namespaces/:ns/paris?limit=20&offset=0)

```sh
$ go-wrk -n=1000 -c=50 -t=10 http://localhost:2021/api/namespaces/ns/pairs\?limit\=20\&offset\=0
==========================BENCHMARK==========================
URL:				http://localhost:2021/api/namespaces/ns/pairs?limit=20&offset=0

Used Connections:		50
Used Threads:			10
Total number of calls:		1000

===========================TIMINGS===========================
Total time passed:		1.41s
Avg time per request:		65.52ms
Requests per second:		710.10
Median time per request:	58.57ms
99th percentile time:		218.70ms
Slowest time for request:	351.00ms

=============================DATA=============================
Total response body sizes:		432000
Avg response body per request:		432.00 Byte
Transfer rate per second:		306765.24 Byte/s (0.31 MByte/s)
==========================RESPONSES==========================
20X Responses:		1000	(100.00%)
30X Responses:		0	(0.00%)
40X Responses:		0	(0.00%)
50X Responses:		0	(0.00%)
Errors:			0	(0.00%)
```

### API (/api/namespaces/:ns/containers/:key) benchmark result

<details>
	<summary>`curl http://localhost:2021/api/namespaces/ns/containers/container-1` </summary>
```json
{
  "errcode": 0,
  "errmsg": "success",
  "data": {
    "key": "container-1",
    "namespace": "ns",
    "checkSum": "24ac0faa8b9e487b056bd50eb922f2b96a5100fbb746f403015fa7f18ae03d4b",
    "fields": [
      {
        "key": "f",
        "fieldType": 1,
        "value": {
          "key": "f",
          "value": 1.123,
          "datatype": 3,
          "namespace": "ns"
        }
      },
      {
        "key": "i",
        "fieldType": 1,
        "value": {
          "key": "i",
          "value": 123,
          "datatype": 1,
          "namespace": "ns"
        }
      },
      {
        "key": "list_basic",
        "fieldType": 2,
        "value": [
          {
            "key": "i",
            "value": 123,
            "datatype": 1,
            "namespace": "ns"
          },
          {
            "key": "f",
            "value": 1.123,
            "datatype": 3,
            "namespace": "ns"
          },
          {
            "key": "i",
            "value": 123,
            "datatype": 1,
            "namespace": "ns"
          },
          {
            "key": "b",
            "value": true,
            "datatype": 4,
            "namespace": "ns"
          }
        ]
      },
      {
        "key": "b",
        "fieldType": 1,
        "value": {
          "key": "b",
          "value": true,
          "datatype": 4,
          "namespace": "ns"
        }
      },
      {
        "key": "d",
        "fieldType": 1,
        "value": {
          "key": "dict",
          "value": {
            "df": 1.123,
            "di": 123,
            "ds": "string"
          },
          "datatype": 6,
          "namespace": "ns"
        }
      },
      {
        "key": "dict",
        "fieldType": 3,
        "value": {
          "b": {
            "key": "b",
            "value": true,
            "datatype": 4,
            "namespace": "ns"
          },
          "dict": {
            "key": "dict",
            "value": {
              "df": 1.123,
              "di": 123,
              "ds": "string"
            },
            "datatype": 6,
            "namespace": "ns"
          },
          "f": {
            "key": "f",
            "value": 1.123,
            "datatype": 3,
            "namespace": "ns"
          },
          "i": {
            "key": "i",
            "value": 123,
            "datatype": 1,
            "namespace": "ns"
          },
          "s": {
            "key": "s",
            "value": "string",
            "datatype": 2,
            "namespace": "ns"
          }
        }
      }
    ]
  }
}
```
</details>

```sh
$ go-wrk -n=1000 -c=40 -t=10 http://localhost:2021/api/namespaces/ns/containers/container-1
==========================BENCHMARK==========================
URL:				http://localhost:2021/api/namespaces/ns/containers/container-1

Used Connections:		40
Used Threads:			10
Total number of calls:		1000

===========================TIMINGS===========================
Total time passed:		2.02s
Avg time per request:		77.40ms
Requests per second:		495.49
Median time per request:	63.93ms
99th percentile time:		204.64ms
Slowest time for request:	275.00ms

=============================DATA=============================
Total response body sizes:		1196000
Avg response body per request:		1196.00 Byte
Transfer rate per second:		592603.46 Byte/s (0.59 MByte/s)
==========================RESPONSES==========================
20X Responses:		1000	(100.00%)
30X Responses:		0	(0.00%)
40X Responses:		0	(0.00%)
50X Responses:		0	(0.00%)
Errors:			0	(0.00%)
```
