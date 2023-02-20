#!/bin/bash

while IFS="," read -r label ip ; do
  curl -X POST "http://${ip}/kill"
  screen -S "$label" -X quit
done < started_nodes.csv

rm started_nodes.csv
