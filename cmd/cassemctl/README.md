## cassemctl

Helps user to initialize `cassem`'s config file and persistence.

### Get Started

Following tells how to initialize the cassem.

#### 1. Generate the default config file

```sh
cassemctl genconf -c CONFIG_PATH
```

Now, you need to modify the config file as you want.

#### 2. Initialize Persistence

```sh
cassemctl -c YOUR_CONF_FILE init persistence 
```

#### 3. Add a root user for your `cassem`, so you can access to the server.

```sh
cassemctl -c YOUR_CONF_FILE add user  
```

Now, you can try to start `cassemd`: [README](../cassemd/README.md)

### Others

You can also add a namespace, container and pair by `cassemctl`, but make sure that the `cassemd`
has been started successfully.

```sh
cassemctl -c YOUR_CONF_FILE add ns --key KEY --data `{}`

cassemctl -c YOUR_CONF_FILE add pair --ns NS --key KEY --data `{}`

cassemctl -c YOUR_CONF_FILE add/del container --ns NS --key KEY --data `{}`
```