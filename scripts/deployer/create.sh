#!/bin/bash

topic=$1
mode=$2
consistency=$3
counter=0
first_ip=''
first_label=''

# setting topic if not set by args
if [[ "$topic" == "" ]]
then
  topic='test'
fi

# setting mode if not set by args
if [[ "$mode" == "s" ]]
then
  mode='single-queue'
else
  mode='multiple-queue'
fi

# creating invitations for each member (except for the first) and
# init corresponding oob protocol via first member
while IFS="," read -r label ip pub ; do
  if [[ $counter == 0 ]]
  then
    counter=$((counter+1))
    first_ip=$ip
    continue
  fi

  inv=$(curl -X GET "http://${ip}/inv")
  curl -X POST --data-raw "${inv}" "${first_ip}/oob"
done < started_nodes.csv

counter=0
while IFS="," read -r label ip pub ; do
  # setting boolean variable for publisher/subscriber role
  is_pub=false
  if [[ "$pub" == "pub" ]]
  then
    is_pub=true
  fi

  # first member creates the group
  if [[ $counter == 0 ]]
  then
    data='{"topic": "'"$topic"'", "publisher": '$is_pub', "consistency": "'"$consistency"'", "mode": "'"$mode"'"}'
    curl -X POST --header 'Content-Type: application/json' --data-raw "$data" "${ip}/create"

    counter=$((counter+1))
    first_label=$label
    continue
  fi

  # each member requests to join from the first member
  data='{"topic": "'"$topic"'", "acceptor": "'"$first_label"'", "publisher": '$is_pub'}'
  curl -X POST --header 'Content-Type: application/json' --data-raw "$data" "${ip}/join"
done < started_nodes.csv
