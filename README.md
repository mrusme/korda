# Korda

[![SEGV 
LICENSE](https://img.shields.io/static/v1?label=SEGV%20LICENSE&message=1.1&labelColor=0060A8&color=ffffff)](https://xn--gckvb8fzb.com/segv/)

Korda is a drop-in replacement for [Immich][immich] built in Go that uses SQLite
and can run in low-end/constrained environments. It supports the official Immich
apps for Android and iOS.

[<img src="https://xn--gckvb8fzb.com/images/chatroom.png" width="275">](https://xn--gckvb8fzb.com/contact/)

Korda is a single Go binary that keeps its data in one SQLite file and one media
folder, and it uses a simple Valkey/Redis server for its job queue. It's meant
for people who want to use Immich but without a heavy PostgreSQL, a container
that gobbles up memory, or the dumpster fire that is NPM.

Korda implements pretty much all operations of Immich's core API (currently at
version 3.3.3). At the moment, however, it has no web interface (besides the
page of a shared link) and I haven't spent any time/effort on getting Immich's
own web app to work with it. The long term plan, however, is to build an
independent web interface that is more lightweight, snappy and works
[without JavaScript](https://disable-javascript.org).

[immich]: https://immich.app

## Requirements

- Go, if you want to build from source.
- A Valkey or Redis server for the job queue.
- `ffmpeg` and `ffprobe`, optional, for video thumbnails, video lengths and
  transcodes. Korda uses the H.264 encoder `libx264`, or `libopenh264` when
  `libx264` is missing, and the filters `zscale` and `tonemap` for HDR videos
  when they exist. Without `ffmpeg`, videos are stored and served but get no
  thumbnails and no transcodes. iPhones record videos in HEVC by default, so
  `ffmpeg` also needs an HEVC decoder, which `ffmpeg -decoders` lists as `hevc`.
  Fedora's `ffmpeg-free` doesn't have that, and with it the thumbnail and
  transcode jobs of these videos fail. Korda logs a warning at startup when its
  `ffmpeg` has no HEVC decoder.

## Building

```sh
make build
```

The binary is `build/korda`, built without cgo, so it can be copied to any
machine of the same platform. For a Raspberry Pi with a 64-bit system, build
with `make build GOOS=linux GOARCH=arm64`. `./build/korda -v` prints the
version, the commit and the Immich version that it reports.

Two features are build tags, off by default because of their licenses, see
_Build tags_ below: `make build TAGS=heic,tzf` includes both.

### Build tags

Individual features can be enabled using `make build TAGS=...`, e.g.
`make build TAGS=heic`, or `make build TAGS=heic,tzf`.

#### HEIC

`heic` decodes HEIC, the format of iPhone photos, with [gen2brain/h265][h265], a
decoder in pure Go, for their thumbnails and previews. When built without it
Korda accepts, stores and serves HEIC files and reads their metadata, but
generates no thumbnail or preview of them.

HEIC compresses with HEVC, which is covered by patents and the decoder's README
states that it "gives you no special rights on the HEVC patents" and that "if
you distribute or use this software you may need a licence" from the patent
holders, so only build with `heic` if that's acceptable to you.

#### TZF

`tzf` finds the time zone of a photo's GPS position when its metadata has no
time zone offset, using [ringsaturn/tzf][tzf]. Without it, the time of such a
photo is read as UTC, unless its GPS time gives the offset.

tzf's code is under the MIT license and its "Anti CSDN License", and the time
zone boundaries it embeds come from [tzf-dist][tzf-dist] under the [ODbL][odbl].

**Note:** Building with `tzf` increases the build (size-wise) significantly (~9
MB larger).

[h265]: https://github.com/gen2brain/h265
[tzf]: https://github.com/ringsaturn/tzf
[tzf-dist]: https://github.com/ringsaturn/tzf-dist
[odbl]: https://opendatacommons.org/licenses/odbl/

## Configuration

The Korda configuration is located at `/etc/korda.toml` unless `-c` specifies a
different path (as in `-c "file:///srv/korda/korda.toml"`).
[`korda.toml`](korda.toml) in this repository is the configuration for
development.

The server settings that an Immich admin normally changes in Immich's web app
come from this file and from Immich's defaults in Korda. Changes through the API
are rejected at the moment.

## Running

```sh
./build/korda -c "file:///etc/korda.toml"
```

On startup korda migrates the database, creates its folders in the media folder,
and starts the web server, the job worker and the nightly jobs. The service
stops on `SIGINT` or `SIGTERM`.

If you only want to migrate the database you can run it with the `-migrate`
flag, which only runs the migrations and exits.

To create accounts, including the first admin account, use the `-create-user`
flag:

```sh
./build/korda -c "file:///etc/korda.toml" \
  -create-user '{"email":"you@example.com","name":"You","password":"...","admin":true}'
```

**Note:** The first user **has to be an admin**. The rest can be created with
`"admin":false`, or from `POST /api/admin/users` with an admin's API key.

### Apps

In the app, enter Korda's address as the server, such as
`http://192.168.1.10:2283` or `https://photos.example.com`. The app finds the
API through the `/.well-known/immich` endpoint.

## Backups

Korda copies its database into `backups/` in the media folder every night at
02:00 (server's local time) and keeps the last 14 copies. A copy is named like
Immich's backups, such as
`korda-db-backup-20261027T020000-v0.1.0-sqlite3.53.4.sql.gz`, but it's a gzipped
SQLite database, not SQL, despite the `.sql.gz` extension. The sole reason why
this naming was kept is because Immich's backup routes require it.

The backup settings can be configured in the configuration. An admin can also
start a backup and list, download, upload and delete backups through Immich's
API under `/api/admin/database-backups`. A failed backup creates a notification
for the admin. With the backups in the media folder, a copy of the media folder
to another disk holds the photos and the database as of the last backup.

To restore a backup, stop korda and run:

```sh
./build/korda -c "file:///etc/korda.toml" \
  -restore korda-db-backup-20261027T020000-v0.1.0-sqlite3.53.4.sql.gz
```

A file name without a directory is looked up in `backups/`, and a plain copy of
the SQLite file works as well. Korda checks that the file is an intact Korda
database that isn't newer than the binary, keeps the current database as
`korda.db.before-restore-<time>`, puts the backup in its place and exits. The
next start then migrates the database if it's older.

**Warning:** The restore fails while Korda is running. Also, in case it wasn't
obvious enough: Immich's own backups can't be restored into Korda, and neither
can Immich's web app restore Korda's backups.

**Note:** A restore ends every session, so each app has to sign in again. The
app then deletes what it synced from Korda, syncs from scratch with its backup
setting unchanged, and re-uploads the photos that were uploaded after the backup
and are still on the phone. The files of those earlier uploads **stay in the
media folder without belonging to any photo**, because Korda doesn't remove them
(as of right now).

## Reverse proxy

Korda serves plain HTTP, which works fine for the Immich apps. For HTTPS,
however, a reverse proxy such as Caddy or nginx is needed in front of it, which
does the SSL offloading. The proxy can serve Korda at the root of a domain or
subdomain, but not under a path, since the Immich apps expect `/api`,
`/.well-known/immich` and the shared link pages at the root.

Behind a proxy, every request reaches Korda from the proxy's address unless
Korda gets the visitor's address from the `X-Forwarded-For` header.

**Warning:** Without that header, the login rate limit counts **all visitors as
one** and a few wrong password entries from any single user might keep everyone
from signing in for up to a minute. In that case the log shows the proxy's
address for every request.

For a proxy on the same machine use a configuration like this:

```toml
[Web]
BindIP             = "127.0.0.1"
ProxyHeader        = "X-Forwarded-For"
EnableIPValidation = true
TrustProxy         = true
TrustLoopback      = true
TrustProxies       = []
```

Korda reads the header only from a trusted proxy, so you either need to trust
the loopback address (`TrustLoopback`) or specify the addresses
(`TrustProxies`)in case the proxy is on another machine/container/pod.

With `EnableIPValidation`, Korda reads the header from the right and takes the
first address that isn't a trusted proxy, so a visitor can't choose an address
by sending an `X-Forwarded-For` of their own. Without it, Korda uses the
header's whole value, which a visitor can change with every request to get past
the limit.

List only the proxy's own address in `TrustProxies`, not whole private ranges,
since every address in it may set the header, and keep `BindIP` at `127.0.0.1`
when the proxy runs on the same machine, so that nobody reaches korda around the
proxy.

From a trusted proxy, Korda also reads `X-Forwarded-Proto` and
`X-Forwarded-Host`, which decide whether the cookie of a shared link is marked
`Secure` and, without `[Server] ExternalDomain`, the address in the preview of a
shared link.

### Caddy

Caddy sets `X-Forwarded-For` to the visitor's address, ignores the
`X-Forwarded-For` a visitor sends, adds `X-Forwarded-Proto` and
`X-Forwarded-Host`, and passes websockets through:

```
photos.example.com {
    reverse_proxy 127.0.0.1:2283
}
```

### nginx

nginx needs the headers, the websocket upgrade, large uploads and long timeouts
set explicitly. This follows Immich's example configuration:

```nginx
server {
    server_name photos.example.com;

    client_max_body_size 50000M;
    proxy_request_buffering off;
    client_body_buffer_size 1024k;

    proxy_set_header Host              $host;
    proxy_set_header X-Real-IP         $remote_addr;
    proxy_set_header X-Forwarded-For   $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;

    proxy_http_version 1.1;
    proxy_redirect     off;

    proxy_read_timeout 600s;
    proxy_send_timeout 600s;
    send_timeout       600s;

    location / {
        proxy_pass       http://127.0.0.1:2283;
        proxy_set_header Upgrade    $http_upgrade;
        proxy_set_header Connection "upgrade";
    }
}
```

`$proxy_add_x_forwarded_for` appends the visitor's address to the header the
visitor sent, which is safe with `EnableIPValidation`, since korda reads the
header from the right.

## Limitations

Korda has no machine learning as of right now, so there's no face recognition,
smart search, OCR or duplicate detection, and no reverse geocoding, so photos
have no places and Korda reports the map as off.

Also, OAuth, email, external libraries, the storage template, real-time video
transcoding, workflows, plugins, maintenance mode and integrity reports aren't
available either. Korda doesn't split the motion photos of Samsung phones and
makes no thumbnails of AVIF, JPEG XL, SVG, JPEG 2000 and CR3 files.

These are features that might come in future iterations but as of right now
don't expect these to work.

## Developing

See [`DEVELOPMENT.md`](DEVELOPMENT.md).

## License

Copyright © 2026 [マリウス](https://xn--gckvb8fzb.com)

Korda is released under Version 1.1 of the
[SEGV License](https://xn--gckvb8fzb.com/segv/), whose full text is in the
[LICENSE](LICENSE) file.
