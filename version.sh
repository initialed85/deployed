#!/bin/bash

set -e

echo "build $(git rev-parse --verify HEAD) @ $(date)" > VERSION
