#!/bin/bash

for node in $(cat started_nodes.txt); do
#    if [[ $counter == target ]]
#    then
#      break
#    fi

    curl -X POST "http://${node}/kill"
#    counter=$((counter+1))
done

rm started_nodes.txt
