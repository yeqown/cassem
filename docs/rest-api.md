## RESTful API

This document describes all Restful APIs in `cassem`.

### Contents

[Response convention](#Response-convention)

[Error Code convention](#Error-Code convention)
 
[1. Namespace](#1-Namespace)

[2. Namespace](#2-Containers)

[3. Namespace](#3-Pairs)

[4. Datatypes](#4-Datatypes)


### Response convention

SUCCESS:

> `data` could be variable type, it depends on specific API.

```json
{
  "errcode": 0,
  "errmsg": "success",
  "data": null
}
```

FAILED:

```json
{
  "errcode": -1,
  "errmsg": "failed reason"
}
```

### Error Code convention

> All success responses will take `StatusOK` (200), otherwise BadRequest or InternalServerError will be returned.
> 

errcode     |description
------------|-----------
0           |success
-1          |failed

### 1 Namespace

#### 1-1 create namespace

create a namespace in cassemd.

Path: `POST /api/namespaces/:{ns}`

Parameters:

field    |type       |description   | position
---------|-----------|--------------|----
ns       |string     |namespace key | PATH

Response:

```json
{
  "errcode": 0,
  "errmsg": "success"
}
```


#### 1-2 paging namespaces

paging namespaces.

Path: `GET /api/namespaces`

Parameters:

field    |type       |description                   | position
---------|-----------|------------------------------|----
limit    |int        |offset to paging, default: 99 | QUERY
offset   |int        |offset to paging, default: 0  | QUERY
key      |string     |namespace key                 | QUERY

Response:

```json
{
  "errcode": 0,
  "errmsg": "success",
  "data": [
    "ns"
  ]
}
```

### 2 Containers

#### 2-1 paging containers

paging containers in cassemd.

Path: `GET /api/namespaces/:{ns}/containers`

Parameters:

field    |type       |description                   | position
---------|-----------|------------------------------|----
ns       |string     |namespace                     | PATH
limit    |int        |offset to paging, default: 10 | QUERY
offset   |int        |offset to paging, default: 0  | QUERY
key      |string     |container key                 | QUERY

Response:

```json
{
  "errcode": 0,
  "errmsg": "success",
  "data": {
    "containers": [
      {
        "key": "del-container-test",
        "namespace": "ns",
        "checkSum": "cb7df8321d9d1b1280731a8fe67bdcd6767a10bfed2b5433e1c8bc6cec8b5804",
        "fields": [
          {
            "key": "dict_kv",
            "fieldType": 1
          },
          {
            "key": "d_dict",
            "fieldType": 3
          },
          {
            "key": "float_kv",
            "fieldType": 1
          },
          {
            "key": "bool_kv",
            "fieldType": 1
          }
        ]
      },
      {
        "key": "container-1",
        "namespace": "ns",
        "checkSum": "24ac0faa8b9e487b056bd50eb922f2b96a5100fbb746f403015fa7f18ae03d4b",
        "fields": [
          {
            "key": "i",
            "fieldType": 1
          },
          {
            "key": "b",
            "fieldType": 1
          },
          {
            "key": "d",
            "fieldType": 1
          },
          {
            "key": "list_basic",
            "fieldType": 2
          },
          {
            "key": "dict",
            "fieldType": 3
          },
          {
            "key": "f",
            "fieldType": 1
          }
        ]
      }
    ],
    "total": 2
  }
}
```

#### 2-2 get container

get container in detail.

Path: `GET /api/namespaces/:{ns}/containers/{container}`

Parameters:

field    |type       |description                   | position
---------|-----------|------------------------------|----
ns       |string     |namespace                     | PATH
container|string     |container key                 | PATH

Response:

```json
{
  "errcode": 0,
  "errmsg": "success",
  "data": {
    "key": "del-container-test",
    "namespace": "ns",
    "checkSum": "cb7df8321d9d1b1280731a8fe67bdcd6767a10bfed2b5433e1c8bc6cec8b5804",
    "fields": [
      {
        "key": "bool_kv",
        "fieldType": 1,
        "value": {
          "key": "b",
          "value": true,
          "datatype": 4,
          "namespace": "ns"
        }
      },
      {
        "key": "d_dict",
        "fieldType": 3,
        "value": {
          "dict_f": {
            "key": "f",
            "value": 1.123,
            "datatype": 3,
            "namespace": "ns"
          },
          "dict_string": {
            "key": "s",
            "value": "string",
            "datatype": 2,
            "namespace": "ns"
          }
        }
      },
      {
        "key": "dict_kv",
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
        "key": "float_kv",
        "fieldType": 1,
        "value": {
          "key": "f",
          "value": 1.123,
          "datatype": 3,
          "namespace": "ns"
        }
      }
    ]
  }
}
```

#### 2-3 create namespace

create a namespace in cassemd.

Path: `POST /api/namespaces/:{ns}/containers/:{container}`

Parameters:

field    |type       |description                   | position
---------|-----------|------------------------------|----
ns       |string     |namespace                     | PATH
container|string     |container key                 | PATH
payload  |JSON       |request JSON body             | BODY

payload example:
```json
{
  "fields": [
    {
      "key": "bool_kv",
      "value": "b",
      "fieldType": 1
    },
    {
      "key": "dict_kv",
      "value": "dict",
      "fieldType": 1
    },
    {
      "key": "d_dict",
      "value": {
        "dict_f": "f",
        "dict_string": "s"
      },
      "fieldType": 3
    },
    {
      "key": "float_kv",
      "value": "f",
      "fieldType": 1
    }
  ]
}
```

Response:

```json
{
  "errcode": 0,
  "errmsg": "success"
}
```

#### 2-4 remove container

remove container from cassemd.

Path: `DELETE /api/namespaces/:{ns}/containers/:{container}`

Parameters:

field    |type       |description                   | position
---------|-----------|------------------------------|----
ns       |string     |namespace                     | PATH
container|string     |container key                 | PATH

Response:

```json
{
  "errcode": 0,
  "errmsg": "success"
}
```

### 3 Pairs

#### 3-1 paging pair 

paging pairs in cassemd.

Path: `GET /api/namespaces/:{ns}/pairs`

Parameters:

field    |type       |description                   | position
---------|-----------|------------------------------|----
ns       |string     |namespace                     | PATH
limit    |int        |offset to paging, default: 10 | QUERY
offset   |int        |offset to paging, default: 0  | QUERY
key      |string     |pair key                      | QUERY

Response:

```json
{
  "errcode": 0,
  "errmsg": "success",
  "data": {
    "pairs": [
      {
        "key": "dict",
        "value": {
          "df": 1.123,
          "di": 123,
          "ds": "string"
        },
        "datatype": 6,
        "namespace": "ns"
      },
      {
        "key": "b",
        "value": true,
        "datatype": 4,
        "namespace": "ns"
      },
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
        "key": "s",
        "value": "string",
        "datatype": 2,
        "namespace": "ns"
      },
      {
        "key": "kv1",
        "value": 32222,
        "datatype": 1,
        "namespace": "ns"
      }
    ],
    "total": 6
  }
}
```

#### 3-2 get pair

get pair in detail.

Path: `GET /api/namespaces/:{ns}/pairs/:{pair}`

Parameters:

field    |type       |description                   | position
---------|-----------|------------------------------|----
ns       |string     |namespace                     | PATH
pair     |string     |pair key                      | PATH

Response:

```json
{
  "errcode": 0,
  "errmsg": "success",
  "data": {
    "key": "dict",
    "value": {
      "df": 1.123,
      "di": 123,
      "ds": "string"
    },
    "datatype": 6,
    "namespace": "ns"
  }
}
```


#### 3-3 create/update pair

get pair in detail.

Path: `POST /api/namespaces/:{ns}/pairs/:{pair}`

Parameters:

field    |type       |description                   | position
---------|-----------|------------------------------|----
ns       |string     |namespace                     | PATH
pair     |string     |pair key                      | PATH
payload  |JSON       |request JSON body             | BODY

payload examples are following:

* int datatype as following

	```json
	{
	  "value": 1,
	  "datatype": 1
	}
	```
* list datatype as following

	```json
	{
	  "value": [
	    1,
	    2,
	    3
	  ],
	  "datatype": 5
	}
	```
  
* int datatype as following
  
	```json
	{
	  "value": {
	    "a": 1,
	    "b": "b"
	   },
	  "datatype": 6
	}
	```

Response:

```json
{
  "errcode": 0,
  "errmsg": "success"
}
```

### 4 Datatypes 

datatype enum|description
-------------|-----------
1            |int
2            |string
3            |float
4            |bool
5            |list
6            |dict