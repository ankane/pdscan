# pdscan

Scan your data stores for unencrypted personal data (PII)

- Last names
- Email addresses
- IP addresses (IPv4)
- Street addresses (US)
- Phone numbers
- Credit card numbers
- Social security numbers
- Dates of birth
- Location data
- OAuth tokens

Uses data sampling and naming, and works with compressed files

:boom: Zero runtime dependencies and minimal database load

[![Build Status](https://github.com/ankane/pdscan/workflows/build/badge.svg?branch=master)](https://github.com/ankane/pdscan/actions)

## Installation

Download the latest version.

- Mac - [x86_64](https://github.com/ankane/pdscan/releases/download/v0.1.3/pdscan_0.1.3_Darwin_x86_64.zip) or [arm64](https://github.com/ankane/pdscan/releases/download/v0.1.3/pdscan_0.1.3_Darwin_arm64.zip)
- Linux - [x86_64](https://github.com/ankane/pdscan/releases/download/v0.1.3/pdscan_0.1.3_Linux_x86_64.zip) or [arm64](https://github.com/ankane/pdscan/releases/download/v0.1.3/pdscan_0.1.3_Linux_arm64.zip)
- Windows - [x86_64](https://github.com/ankane/pdscan/releases/download/v0.1.3/pdscan_0.1.3_Windows_x86_64.zip)

Unzip and follow the instructions below for your data store.

With Homebrew, you can also use:

```sh
brew install ankane/brew/pdscan
```

## Data Stores

- [Elasticsearch](#elasticsearch) [unreleased]
- [Files](#files)
- [MariaDB](#mariadb)
- [MongoDB](#mongodb) [unreleased]
- [MySQL](#mysql)
- [OpenSearch](#opensearch) [unreleased]
- [Postgres](#postgres)
- [Redis](#redis) [unreleased]
- [SQLite](#sqlite)
- [SQL Server](#sql-server) [unreleased]
- [S3](#s3)

### Elasticsearch

```sh
pdscan elasticsearch+http://user:pass@host:9200
```

> Requires Elasticsearch 7 at the moment

For HTTPS, use `elasticsearch+https://`.

You can also specify indices.

```sh
pdscan elasticsearch+http://user:pass@host:9200/index1,index2
```

Wildcards are also supported.

```sh
pdscan "elasticsearch+http://user:pass@host:9200/index*"
```

### Files

```sh
pdscan file://path/to/file.txt
```

You can also specify a directory.

```sh
pdscan file://path/to/directory
```

For absolute paths, use `file:///`.

### MariaDB

```sh
pdscan mariadb://user:pass@host:3306/dbname
```

### MongoDB

```sh
pdscan mongodb://user:pass@host:27017/dbname
```

### MySQL

```sh
pdscan mysql://user:pass@host:3306/dbname
```

### OpenSearch

```sh
pdscan opensearch+http://user:pass@host:9200
```

For HTTPS, use `opensearch+https://`.

You can also specify indices.

```sh
pdscan opensearch+http://user:pass@host:9200/index1,index2
```

Wildcards are also supported.

```sh
pdscan "opensearch+http://user:pass@host:9200/index*"
```

### Postgres

```sh
pdscan postgres://user:pass@host:5432/dbname
```

If your connection doesn’t use SSL, append to the URI:

```
?sslmode=disable
```

For best sampling, enable the [tsm_system_rows](https://www.postgresql.org/docs/current/tsm-system-rows.html) extension (ships with Postgres 9.5+).

```sql
CREATE EXTENSION tsm_system_rows;
```

### Redis

```sh
pdscan redis://user:pass@host:6379/db
```

Only scans string keys at the moment

### SQLite

```sh
pdscan sqlite:/path/to/dbname.sqlite3
```

### SQL Server

```sh
pdscan "sqlserver://user:pass@host:1433?database=dbname"
```

### S3

```sh
pdscan s3://bucket/path/to/file.txt
```

> Requires `s3:GetObject` permission

You can also specify a prefix by ending with a `/`.

```sh
pdscan s3://bucket/path/to/directory/
```

> Requires `s3:ListBucket` and `s3:GetObject` permissions

## Options

Show data found

```sh
pdscan --show-data
```

Show low confidence matches

```sh
pdscan --show-all
```

Change sample size

```sh
pdscan --sample-size 50000
```

Specify number of processes to use (defaults to 1)

```sh
pdscan --processes 4
```

## History

View the [changelog](https://github.com/ankane/pdscan/blob/master/CHANGELOG.md)

## Contributing

Everyone is encouraged to help improve this project. Here are a few ways you can help:

- [Report bugs](https://github.com/ankane/pdscan/issues)
- Fix bugs and [submit pull requests](https://github.com/ankane/pdscan/pulls)
- Write, clarify, or fix documentation
- Suggest or add new features

To get started with development:

```sh
git clone https://github.com/ankane/pdscan.git
cd pdscan
make test
```
