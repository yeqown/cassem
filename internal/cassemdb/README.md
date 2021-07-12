## cassemdb

provide my RAFT and Repository API and implementation.


```plaintext
.
├── api       export function to other client.
├── app       (domain / infras)
├── domain    (infras)
├── infras    (no package could be imported)
```

## Features

- [x] distributed system based single-raft.
- [x] TTL
- [x] lock operation based distributed system
- [x] watching mechanism (directory or key)
- [x] support operations set (key, dir) / unset (key / dir) / get (key) / range (dir)
- [ ] dynamic add or remove node API.

### Operations prototype

```plaintext
# set a kv or create a directory
set(key, val, isDir, overwrite, ttl)

# use as lock, presudo code like this:
if err = set(k, v, false, overwrite=false, ttl); err != ErrKeyExists {
    // mean key has been set.
}

# remove a kv or a directory, if target key is not found, no error will returned.
unset(key, isDir)

# get a kv
get(key)

# range a directory, return all key, value and type (kv, dir) of it,
# if dirKey is a kv type, ErrNotDir will be returned, at meantime dir type item has no value
# in result.
range(dirKey)

# only works while target key is a kv type and it's ttl is bigger than 0. 
expire(key)

# shows how long will the key alive.
ttl(key)
```

### Error code

```plaintext
ErrNotFound
ErrNotDir
ErrKeyExists 
```