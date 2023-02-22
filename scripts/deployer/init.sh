#!/bin/bash

buf=$1
num_nodes=$2
mode=$3
user=$4
key_path=$5
counter=0

# read params from the file (port, pubport etc)
declare -a labels ips ports pub_ports mock_ports pubs
while IFS="," read -r label ip port pub_port mock_port pub ; do
  labels+=("$label")
  ips+=("$ip")
  ports+=("$port")
  pub_ports+=("$pub_port")
  mock_ports+=("$mock_port")
  pubs+=("$pub")
done < agents.csv

go build ../..

for label in "${labels[@]}"; do
  if [[ counter -eq num_nodes ]]
  then
    break
  fi

  screen -d -m -S "$label" ./didcomm-prober -label="${labels[$counter]}" -port="${ports[$counter]}" -pub="${pub_ports[$counter]}" -mock_port="${mock_ports[$counter]}" -buf="$buf" -v -sync "$mode"
#  screen -d -m -S "$label" ssh -i "$key_path" "$user@${ips[$counter]}" "cd agent/ && ./didcomm-prober -label=${labels[$counter]} -port=${ports[$counter]} -pub=${pub_ports[$counter]} -mock_port=${mock_ports[$counter]} -buf="$buf" "$mode" -v"

  node="${ips[$counter]}:${mock_ports[$counter]}"
  echo "$label - $node started"
  echo "$label,$node,${pubs[counter]}" >> started_nodes.csv
  counter=$((counter+1))
done
