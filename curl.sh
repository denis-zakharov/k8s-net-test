#!/bin/bash

curl -v http://localhost:8080/svc -H "Content-Type: application/json" \
-d '{"svcURL": "http://localhost:8080/ping", "count": 1000}'

curl -v http://localhost:8080/direct -H "Content-Type: application/json" \
-d '[{"hostname": "pod1", "addrs": ["192.168.1.1"]}, {"hostname": "pod2", "addrs": ["fd00:10:244::8"]}]'

curl -v http://localhost:8080/direct -H "Content-Type: application/json" \
-d '[{"hostname": "localhost", "addrs": ["127.0.0.1", "::1"]}]'
