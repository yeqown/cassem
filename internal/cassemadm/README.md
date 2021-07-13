## cassemadm

Admin server to manage all data in cassem.


### Features

- [x] Config CURD and API
- [ ] Operation Logs 
- [ ] Multiple Content Type support plugin mode (JSON / TOML / Plain Text / INI)
- [ ] gray released
- [ ] 


***ContentType Plugin***
```go
type ContentTypePlugin interface {
	// ContentType return an unique identifier string for plugin
	// For example: JSON => content-type:json
	ContentType() string
	Encode(v interface{}) ([]bytes, error)
	Decode(data []bytes, v *Element) error
}
```