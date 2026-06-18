## Documentation

- [Key storage structure](key-storage.md)
- [Key storage issues](key-storage-issues.md)

### Build Docker image

```sh
# path/to/cassem
docker build -t yeqown/cassemdb:v0.9.0-rc1 -f ./.deploy/dockerfiles/cassemdb.Dockerfile .
docker build -t yeqown/cassemadm:v0.9.0-rc1 -f ./.deploy/dockerfiles/cassemadm.Dockerfile .
docker build -t yeqown/cassemagent:v0.9.0-rc1 -f ./.deploy/dockerfiles/cassemagent.Dockerfile .
```