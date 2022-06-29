#!/bin/sh -e

docker-compose build
docker-compose run --rm integration-tests
docker-compose down
