#!/bin/bash

go build
mv tester ../deployer/
cd ../deployer/ || return
./tester join 3000
bash term.sh
rm tester didcomm-prober
