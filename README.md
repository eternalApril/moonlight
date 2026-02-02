# 🌙 Moonlight

![Go Version](https://img.shields.io/badge/go-1.25-blue)
![License](https://img.shields.io/badge/license-Apache-green)
![Status](https://img.shields.io/badge/status-active-success)

Moonlight is a lightweight, in-memory database engine designed to demonstrate high-concurrency patterns in Go.
It implements the RESP (Redis Serialization Protocol), making it compatible with existing Redis clients (like
redis-cli).

## ️ Supported Commands
Moonlight currently supports commands:

| Command   | Description                                      | Supported Flags                                    |
|:----------|:-------------------------------------------------|:---------------------------------------------------|
| `COMMAND` | Return an array with details about every command | `COUNT`                                            |
| `PING`    | Check server health                              | -                                                  |
| `GET`     | Get value by key                                 | -                                                  |
| `SET`     | Set key to value                                 | `NX`, `XX`, `EX`, `PX`, `EXAT`, `PXAT`, `KEEPTTL`  |
| `DEL`     | Delete one or more keys                          | -                                                  |
| `TTL`     | Get remaining time (sec)                         | -                                                  |
| `PTTL`    | Get remaining time (ms)                          | -                                                  |
| `PERSIST` | Remove the existing timeout on key               | -                                                  |
| `SAVE`    | Save data to disk                                | -                                                  |
| `BGSAVE`  | Save data to disk (background process)           | -                                                  |

## Installation & Usage

### Option 1: Docker Compose

```bash
# Clone and run
docker-compose up --build
```

### Option 2: Docker

```bash
# 1. Build the image
docker build -t moonlight .

# 2. Run container
docker run -p 6380:6380 --name moonlight -d moonlight

# 3. Connect using redis-cli
redis-cli -p 6380
```

### Option 3: Running Locally

```bash
# Clone and run
go mod download
go run main.go
```

## Configuration

Moonlight can be configured via a `config.yml` file in the root directory OR via Environment Variables. Environment variables take precedence.

| YAML Key                   | Env Variable                    | Default          | Description                                                                   |
|:---------------------------|:--------------------------------|:-----------------|:------------------------------------------------------------------------------|
| `server.port`              | `MOONLIGHT_SERVER_PORT`         | `6380`           | TCP Port to listen on                                                         |
| `storage.shards`           | `MOONLIGHT_STORAGE_SHARDS`      | `32`             | Number of map shards (Power of 2)                                             |
| `gc.enabled`               | `MOONLIGHT_GC_ENABLED`          | `true`           | Enable background expiration                                                  |
| `gc.interval`              | `MOONLIGHT_GC_INTERVAL`         | `100ms`          | How often GC runs                                                             |
| `gc.sample_per_shard`      | `MOONLIGHT_GC_SAMPLE_PER_SHARD` | `20`             | How many keys GC check in every shard                                         |
| `gc.expand_threshold`      | `MOONLIGHT_GC_EXPAND_THRESHOLD` | `0.25`           | The percentage of expired keys, at which the GC repeats the check immediately |
| `log.level`                | `MOONLIGHT_LOG_LEVEL`           | `debug`          | `debug`, `info`, `warn`, `error`                                              |
| `log.format`               | `MOONLIGHT_LOG_FORMAT`          | `json`           | `json` or `console`                                                           |
| `persistence.aof.enabled`  | `PERSISTENCE_AOF_ENABLED`       | `false`          | Enable AOF persistence                                                        |
| `persistence.aof.filename` | `PERSISTENCE_AOF_FILENAME`      | `appendonly.aof` | Path to file, for AOF persistence, create file if not exist                   |
| `persistence.aof.fsync`    | `PERSISTENCE_AOF_FSYNC`         | `everysec`       | How often to dump data to disk, `everysec`, `always`, `no`                    |
| `persistence.rdb.enabled`  | `PERSISTENCE_RDB_ENABLED`       | `false`          | Enable RDB persistence                                                        |
| `persistence.rdb.filename` | `PERSISTENCE_RDB_FILENAME`      | `dump.rdb `      | Path to file, for RDB persistence, create file if not exist                   |
| `persistence.rdb.interval` | `PERSISTENCE_RDB_INTERVAL`      | `60s`            | How often to dump data to disk                                                |

**Example `config.yml`:**
```yml
server:
  host: "0.0.0.0"
  port: "6380"

storage:
  shards: 32

gc:
  enabled: true
  interval: "100ms"
  sample_per_shard: 20
  expand_threshold: 0.25

log:
  level: "debug"
  format: "json"

persistence:
  rdb:
    enabled: true
    filename: "dump.rdb"
    interval: "60s"
```

## License

Distributed under the Apache License. See `LICENSE` for more information.
