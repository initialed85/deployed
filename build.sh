#!/bin/bash

set -e

go generate -x .

go build -x -o ./deployed -trimpath .
