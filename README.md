PG Delta Stream
===============
Introduction
------------
PG Delta Stream uses PostgreSQL's logical decoding feature to stream table changes over websockets. 

PostgreSQL logical decoding allows the streaming of the Write Ahead Log (WAL) via an output plugin (wal2json) from a logical replication slot.

Overview of PGDeltaStream
-------------------------
The workflow to use this application is split into 3 phases:

1. Init: Create a replication slot

2. Snapshot data: Get data from the snapshot

3. Stream: Stream the WAL changes from the consistent point over a websocket connection

Requirements
------------
- PostgreSQL 10 running on port `5432` with the [wal2json](https://github.com/eulerto/wal2json) plugin

Configuration
-------------
To use the logical replication feature, set the following parameters in `postgresql.conf`:

```
wal_level = logical
max_replication_slots = 4
```

Further, add this to `pg_hba.conf`:

```
host    replication     all             127.0.0.1/32            trust
```

Restart the postgresql service.

Launching
---------
Launch the application server on localhost port 12312:
```
go run hasuradb localhost localhost:12312
```

Usage
-----

**Init**

Call the `/v1/init` endpoint to create a replication slot and get the slot name. 

```bash
$ curl localhost:12312/v1/init 
{"slotName":"delta_face56"}
```

Keep note of this slot name to use in the next phases.

**Get snapshot data**

To get data from the snapshot, make a POST request to the `/v1/snapshot/data` endpoint with the table name, offset and limit:
```
curl -X POST \
  http://localhost:12312/v1/snapshot/data \
  -H 'content-type: application/json' \
  -d '{"table": "test_table", "offset":0, "limit":10}'
```

Note that only the data upto the time the replication slot was created will be available in the snapshot. 

**Stream changes over websocket***

Connect to the websocket endpoint to `/v1/lr/stream` along with the slot name to start streaming the changes:

```
ws://localhost:12312/v1/lr/stream?slotName=delta_face56
```