#!/usr/bin/env python3

from matplotlib import pyplot as plt
import pandas as pd
import csv

def readData(name):
    topics = []
    init_sizes = []
    modes = []
    consistency = []
    ordering = []
    bufs = []
    latency = []
    with open(name) as file:        
        reader = csv.reader(file)
        line_index = 0
        for row in reader:
            if line_index > 0:
                topics.append(row[0])
                init_sizes.append(int(row[1]))
                modes.append(row[2])
                consistency.append(row[3])
                ordering.append(row[4])
                bufs.append(int(row[5]))
                latency.append(float(row[6]))
            line_index += 1
    return topics, init_sizes, modes, consistency, ordering, bufs, latency

def plot(filtr, topics, init_sizes, bufs, latency):
    tmp_sizes = []
    tmp_lat = []
    for i in range(len(topics)):
        if topics[i] == filtr:
            tmp_sizes.append(init_sizes[i])
            tmp_lat.append(latency[i])
            buf = bufs[i]
            
    plt.plot(tmp_sizes, tmp_lat, label="sq-c-o-topic", color="seagreen")
    plt.legend()
    plt.xlabel('initial group size')
    plt.ylabel('average time (ms)')
    plt.axhline(y=buf, color='r', linestyle='--', label='zmq-buffer')
    
#    plt.savefig('get_line.pdf', bbox_inches="tight")
    plt.show()

topics, init_sizes, modes, consistency, ordering, bufs, latency = readData('../results/join.csv')   
plot('sq-c-o-topic', topics, init_sizes, bufs, latency)

