## Deploy in Kubernetes（unavailable yet）

`cassem` is a stateful application, so we need use `StatefulSet` deployment type in kubernetes.

### Namespace

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: cassemd
```

### Config Map

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: cassemd
  labels:
    app: cassemd
data:
  cassemd.toml: |
    # Apply this config only on the master.
    debug = false
    [persistence]
    [persistence.mysql]
    dsn             = "cassemd:cassemd@tcp(mysql:3306)/cassem?charset=utf8mb4&parseTime=true&loc=Local"
    max_idle        = 10
    max_open        = 100
    max_life_time   = 30
    [server]
    [server.http]
    addr    = ":2021"
```

### StatefulSet YAML

```yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: cassemd
spec:
  selector:
    matchLabels:
      app: cassemd
  serviceName: cassemd
  replicas: 1
  template:
    metadata:
      labels:
        app: cassemd
    spec:
      containers:
      - name: cassemd
        image: yeqown/cassem
        command: ["/bin/sh", "-c"]
        args: ["./cassemd",
	           "--conf=./configs/cassem.toml", 
               "--id=cassemd-$(hostname)", 
               "--raft-base=/etc/cassemd", 
               "--http-listen=0.0.0.0:2021", 
               "--bind=0.0.0.0:3021", 
               "--join=cassemd-0.cassemd.svc.cluster.local"]
        env:
        - name: IGNORE_AUTH
          value: "1"
        ports:
        - name: http-grpc
          containerPort: 2021
        - name: raft
          containerPort: 3021
        volumeMounts:
        - name: raftbase
          mountPath: /etc/cassemd
        - name: conf
          mountPath: /app/cassem/configs
        resources:
          requests:
            cpu: 500m
            memory: 1Gi
        livenessProbe:
          tcpSocket:
            port: 2021
          initialDelaySeconds: 30
          periodSeconds: 10
          timeoutSeconds: 5
        readinessProbe:
          httpGet:
            scheme: HTTP
            path: /api/namespaces
            port: 2021
          initialDelaySeconds: 5
          periodSeconds: 2
          timeoutSeconds: 1
      volumes:
      - name: conf
        configMap:
          name: cassemd

  volumeClaimTemplates:
  - metadata:
      name: data
    spec:
      accessModes: ["ReadWriteOnce"]
      resources:
        requests:
          storage: 10Gi
```

### Service YAML

```yaml
# Headless service for stable DNS entries of StatefulSet members.
apiVersion: v1
kind: Service
metadata:
  name: cassemd
  labels:
    app: cassemd
spec:
  ports:
  - name: cassemd-http
    port: 2021
  clusterIP: None
  selector:
    app: cassemd
```

### deploy a MySQL cluster in k8s

https://kubernetes.io/zh/docs/tasks/run-application/run-single-instance-stateful-application/