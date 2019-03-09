# pdscan

Scan your data stores for unencrypted personal data (PII)

- Last names
- Email addresses
- IP addresses
- Street addresses (US)
- Phone numbers (US)
- Credit card numbers
- Social security numbers
- Dates of birth
- Location data
- OAuth tokens

Uses data sampling and naming, and works with compressed files

:boom: Zero runtime dependencies and minimal database load

[![Build Status](https://travis-ci.org/ankane/pdscan.svg?branch=master)](https://travis-ci.org/ankane/pdscan)

## Installation

Download the latest version

- [Mac](https://github.com/ankane/pdscan/releases/download/v0.1.0/pdscan_0.1.0_Darwin_x86_64.zip)
- [Linux](https://github.com/ankane/pdscan/releases/download/v0.1.0/pdscan_0.1.0_Linux_x86_64.zip)
- [Windows](https://github.com/ankane/pdscan/releases/download/v0.1.0/pdscan_0.1.0_Windows_x86_64.zip)

Unzip and follow the instructions for your data store

Data Stores

- [Files](#files-master)
- [MySQL & MariaDB](#mysql--mariadb)
- [Postgres](#postgres)
- [SQLite](#sqlite)
- [S3](#s3-master)
- [Others](#others)

## Data Stores

### Files [master]

```sh
./pdscan file://path/to/file.txt
```

You can also specify a directory.

```sh
./pdscan file://path/to/directory
```

For absolute paths, use `file:///`.

### MySQL & MariaDB

```sh
./pdscan mysql://user:pass@host:3306/dbname
```

### Postgres

```sh
./pdscan postgres://user:pass@host:5432/dbname
```

If your connection doesn’t use SSL, append to the URI:

```
?sslmode=disable
```

For best sampling, enable the [tsm_system_rows](https://www.postgresql.org/docs/current/tsm-system-rows.html) extension (ships with Postgres 9.5+).

```sql
CREATE EXTENSION tsm_system_rows;
```

### SQLite

```sh
./pdscan sqlite:/path/to/dbname.sqlite3
```

### S3 [master]

```sh
./pdscan s3://bucket/path/to/file.txt
```

> Requires `s3:GetObject` permission

You can also specify a prefix by ending with a `/`.

```sh
./pdscan s3://bucket/path/to/directory/
```

> Requires `s3:ListBucket` and `s3:GetObject` permissions

### Others

Feel free to [submit a PR](https://github.com/ankane/pdscan/pulls)

## Options

Show data found

```sh
./pdscan --show-data
```

Show low confidence matches

```sh
./pdscan --show-all
```

Change sample size

```sh
./pdscan --sample-size 50000
```

Increase parallelism

```sh
./pdscan --processes 4
```

## Roadmap

- Add more data stores (SQL Server, MongoDB, Elasticsearch, Memcached, Redis)
- Improve rules
- Highlight matches
- Add more output formats, like JSON and CSV

## History

View the [changelog](https://github.com/ankane/pdscan/blob/master/CHANGELOG.md)

## Contributing

Everyone is encouraged to help improve this project. Here are a few ways you can help:

- [Report bugs](https://github.com/ankane/pdscan/issues)
- Fix bugs and [submit pull requests](https://github.com/ankane/pdscan/pulls)
- Write, clarify, or fix documentation
- Suggest or add new features

To get started with development and testing:

```sh
git clone https://github.com/ankane/pdscan.git
cd pdscan
dep ensure
make test
```
