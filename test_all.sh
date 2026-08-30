#!/bin/bash

set -e

./test.sh restore
./test.sh

./test.sh e2e-restore
./test.sh

echo -e "\nall done."
