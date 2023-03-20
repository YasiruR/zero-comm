#!/usr/bin/env python3

from matplotlib import pyplot as plt
import pandas as pd
import csv

# add error bars
# draw all in one?

def readData(fileName, topic):
    init_sizes = []
    bufs = []
    latency = []
    with open(fileName) as file:        
        reader = csv.reader(file)
        line_index = 0
        for row in reader:
            if line_index > 0:
                init_sizes.append(int(row[1]))
                bufs.append(int(row[5]))
                latency.append(float(row[6]))
            line_index += 1
    return init_sizes, bufs, latency

def plot(filtr, init_sizes, buf, latency):
    sizes = []
    avg_lat_list = []
    tmp_size = init_sizes[0]
    total = 0.0
    counter = 0
    for i in range(len(latency)):
        if tmp_size != init_sizes[i] or i == len(latency)-1:
            avg_lat_list.append(total/float(counter))
            sizes.append(tmp_size)
            tmp_size = init_sizes[i]
            total = latency[i]
            counter = 1
        else:
            total += latency[i]
            counter += 1
          
    plt.plot(sizes, avg_lat_list, label=filtr, color="seagreen")
    plt.legend()
    plt.xlabel('initial group size')
    plt.ylabel('average time (ms)')
    plt.axhline(y=buf, color='r', linestyle='--', label='zmq-buffer')
    
#    plt.savefig('get_line.pdf', bbox_inches="tight")
    plt.show()

topic = 'sq-c-o-topic'
init_sizes, bufs, latency = readData('../results/join.csv', topic)
plot(topic, init_sizes, bufs[0], latency)

