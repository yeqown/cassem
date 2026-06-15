## cassemadm

Admin dashboard for cassem manager.

### Web UI dev mode

For local Web UI development, run the frontend as a standalone Vite server on its own `IP:PORT` instead of relying on embedded `/ui/` assets.

```bash
make ui.install
CASSEMADM_API_TARGET=http://127.0.0.1:20218 npm run dev --prefix internal/cassemadm/web -- --port 4173
```

Open `http://<your-ip>:4173/`. The Vite server proxies `/api` requests to `CASSEMADM_API_TARGET`. Embedded `/ui/` assets are still used for `make ui.build` and normal `cassemadm` serving.
