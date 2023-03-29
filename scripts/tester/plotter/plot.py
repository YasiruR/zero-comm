#!/usr/bin/env python3

from matplotlib import pyplot as plt
import numpy as np
import csv

# add error bars
# draw all in one?

# name,initial_size,mode,consistent_join,causally_ordered,init_connected,latency_ms
def readJoinLatency(fileName, topic):
    init_sizes = []
    conctd = []
    latency = []
    with open(fileName) as file:        
        reader = csv.reader(file)
        line_index = 0
        for row in reader:
            if line_index > 0:
                init_sizes.append(int(row[1]))
                conctd.append(row[5])
                latency.append(float(row[6]))
            line_index += 1
    return init_sizes, conctd, latency

def parseJoinLatency(filtrConctd, init_sizes, conctd, latency):
    sizes = []
    avg_lat_list = []
    err_list = []
    
    tmp_size = init_sizes[0]
    tmp_lat_list = []
    for i in range(len(latency)):
        if conctd[i] != filtrConctd:
            continue
        if tmp_size != init_sizes[i]:
            avg_lat_list.append(np.mean(tmp_lat_list))
            err_list.append(np.std(tmp_lat_list))
            sizes.append(tmp_size)
            tmp_size = init_sizes[i]
            tmp_lat_list = [latency[i]]
        else:
            tmp_lat_list.append(latency[i])
    
    avg_lat_list.append(np.mean(tmp_lat_list))
    err_list.append(np.std(tmp_lat_list))
    sizes.append(tmp_size)
    return sizes, avg_lat_list, err_list

def plotJoinLatency(sizes_con, sizes_ncon, avg_lat_list_con, avg_lat_list_ncon, err_list_con, err_list_ncon):
    fig, ax = plt.subplots()
    ax.errorbar(sizes_con, avg_lat_list_con, yerr=err_list_con, ecolor="red", label="connected")
    ax.errorbar(sizes_ncon, avg_lat_list_ncon, yerr=err_list_ncon, ecolor="red", color="green", label="not-connected")
    ax.set_xlabel('initial group size')
    ax.set_ylabel('average time (ms)')
    ax.set_title('Latency for joins')
    plt.legend()
    #    plt.savefig('join_latency.pdf', bbox_inches="tight")
    plt.show()

topic = 'sc-c-o-topic'
init_sizes, conctd, latency = readJoinLatency('../results/join_latency.csv', topic)
sizes_con, avg_lat_list_con, err_list_con = parseJoinLatency('true', init_sizes, conctd, latency)
sizes_ncon, avg_lat_list_ncon, err_list_ncon = parseJoinLatency('false', init_sizes, conctd, latency)
plotJoinLatency(sizes_con, sizes_ncon, avg_lat_list_con, avg_lat_list_ncon, err_list_con, err_list_ncon)
