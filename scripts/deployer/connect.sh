#!/bin/bash

# this script connects the local agent to all the started nodes

limit=$1
mock_port=$2

inv=$(curl -X GET "http://127.0.1.1:${mock_port}/inv")
while IFS="," read -r label ip pub ; do
  if [[ $counter == $limit ]]
  then
    break
  fi

  curl -X POST --data-raw "${inv}" "${ip}/oob"
  counter=$((counter+1))
  echo "connected to ${label}"
done < started_nodes.csv
