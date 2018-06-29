#!/bin/sh

# Create admin
curl --verbose -X PUT -H "X-API-TOKEN: root-token" -d "{\"Token\":\"admin-token\",\"Is\":true}" http://localhost:3000/m/admin
echo

# Create namespace admin
curl --verbose -X PUT -H "X-API-TOKEN: admin-token" -d "{\"Token\":\"ns-admin-token\",\"Is\":true}" http://localhost:3000/m/admin/testing
echo

# Create a put/get token
curl --verbose -X PUT -H "X-API-TOKEN: ns-admin-token" -d "{\"Token\":\"the-token\",\"Put\":true,\"Get\":true,\"Append\":true}" http://localhost:3000/m/token/testing/test.log
echo

# Upload something
curl --verbose -X POST -H "X-API-TOKEN: the-token" -d "Hello, world!" http://localhost:3000/r/testing/test.log
echo

curl --verbose -X PUT -H "X-API-TOKEN: the-token" -d "Bye." http://localhost:3000/r/testing/test.log
echo

# Download something
curl --verbose -X GET -H "X-API-TOKEN: the-token" http://localhost:3000/r/testing/test.log
echo
