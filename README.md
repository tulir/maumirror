# maumirror
A GitHub repo mirroring system using webhooks.

Mirrored to [mau.dev/tulir/maumirror](https://mau.dev/tulir/maumirror) (using maumirror, of course) for CI.

Support room on [Matrix](https://matrix.org): [#maunium:maunium.net](https://matrix.to/#/#maunium:maunium.net)
(bridged to Telegram: [t.me/maunium](https://t.me/maunium))

## Running
### Plain
0. Build with `go build` or download a build from [mau.dev/tulir/maumirror](https://mau.dev/tulir/maumirror/pipelines)
   ([latest build direct link](https://mau.dev/tulir/maumirror/-/jobs/artifacts/master/raw/maumirror?job=build))
1. Copy `example-config.json` to `config.json` and configure
2. Run `./maumirror`

### Docker (compose)
```yaml
version: "3.7"

services:
  maumirror:
    image: dock.mau.dev/tulir/maumirror
    restart: unless-stopped
    volumes:
    - /var/maumirror:/data
    - /etc/maumirror:/config
    - /etc/maumirror/.ssh:/home/maumirror/.ssh
```

0. Install [docker](https://docs.docker.com/install/) and [docker-compose](https://docs.docker.com/compose/install/)
1. Copy `example-config.json` to `/etc/maumirror/config.json` and configure
2. Create `docker-compose.yml` with the content above
3. Ensure the volumes have correct permissions: `sudo chown 29321.29321 -R /etc/maumirror /var/maumirror`
4. Start with `docker-compose up -d`
