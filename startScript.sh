#!/bin/bash

go run /data/gsbckp.go & disown
redis-server /etc/redis/redis.conf -DFOREGROUND