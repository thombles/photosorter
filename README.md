# photosorter

A personal tool for monitoring for new photos synced from a phone via SyncThing and placing them into the real photos directory automatically.

Each photo's modification time is used to determine the year it was taken and it is copied to `${target}/${year}/${filename}`.

A cache of seen files is maintained in `~/.cache` so that files which are deleted from the target directory are not re-copied.

## Usage

Compile:

```
go build
```

Install the binary in a location such as `/usr/local/bin/photosorter`.

Copy the provided systemd template `photosorter.service` to `~/.config/systemd/user/` and edit the source path (`-s`) and target path (`-t`) to match your needs.

Activate the service:

```
systemctl --user enable photosorter
systemctl --user start photosorter
```

Watch output with:

```
journalctl --user-unit photosorter -f
```

## Licence

MIT
