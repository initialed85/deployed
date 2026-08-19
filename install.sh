#!/bin/bash

set -e

./build.sh

go install -x -trimpath .
