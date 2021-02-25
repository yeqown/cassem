## cassemd

To start the `server` and it's daemon process, components in `server` include: 
* `RESTful HTTP`
* `Authorize` middleware
* `coordinator`
* `Cache` middleware
* `Watcher` to watch containers' changes. 
* `Persistence` to persist `cassem's` data.

### Get started

Now, you can start the `cassemd` server as following command:

```sh
# TODO(@yeqown): finish this part.
./cassemd -c CONFIG_FILE start --append-to-cluster="127.0.0.1:2031" --port 2032 --http-port=2022
```

then you'll got:

```sh
# started output

```