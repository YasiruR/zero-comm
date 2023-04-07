#!/usr/bin/env python3

from matplotlib import pyplot as plt
import numpy as np
import csv

# add error bars
# draw all in one?

# join latency functions

def readJoinLatency(fileName):
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
    ax.set_ylabel('average time taken (ms)')
    ax.set_title('Latency results of group-join')
    plt.rc('grid', linestyle="--", color='#C6C6C6')
    plt.legend()
    plt.grid()
    plt.savefig('join_latency.pdf', bbox_inches="tight")
    plt.show()

init_sizes, conctd, latency = readJoinLatency('../results/join_latency.csv')
sizes_con, avg_lat_list_con, err_list_con = parseJoinLatency('true', init_sizes, conctd, latency)
sizes_ncon, avg_lat_list_ncon, err_list_ncon = parseJoinLatency('false', init_sizes, conctd, latency)
plotJoinLatency(sizes_con, sizes_ncon, avg_lat_list_con, avg_lat_list_ncon, err_list_con, err_list_ncon)

# join throughput functions

def readJoinThroughput(fileName, batchSize):
    init_sizes = []
    conctd = []
    latency = []
    with open(fileName) as file:        
        reader = csv.reader(file)
        line_index = 0
        for row in reader:
            if line_index > 0:
                if int(row[6]) != batchSize:
                    continue
                init_sizes.append(int(row[1]))
                conctd.append(row[5])
                latency.append(float(row[7]))
            line_index += 1
    return init_sizes, conctd, latency

def parse(filtrConctd, init_sizes, conctd, latency):
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

def plotJoinThroughput(sizes_con_4, sizes_ncon_4, sizes_con_16, sizes_ncon_16, avg_lat_list_con_4, avg_lat_list_ncon_4, avg_lat_list_con_16, avg_lat_list_ncon_16):
    fig, ax = plt.subplots()   
    plt.plot(sizes_con_4, avg_lat_list_con_4, label="batch=4,connected")
    plt.plot(sizes_ncon_4, avg_lat_list_ncon_4, color="green", label="batch=4,not-connected")
    plt.plot(sizes_con_16, avg_lat_list_con_16, color="purple", label="batch=16,connected")
    plt.plot(sizes_ncon_16, avg_lat_list_ncon_16, color="brown", label="batch=16,not-connected")
    ax.set_xlabel('initial group size')
    ax.set_ylabel('average time taken (ms)')
    ax.set_title('Throughput results of group-join')
    plt.rc('grid', linestyle="--", color='#C6C6C6')
    plt.legend()
    plt.grid()
    plt.savefig('join_throughput.pdf', bbox_inches="tight")
    plt.show()

init_sizes_4, conctd_4, latency_4 = readJoinThroughput('../results/join_throughput.csv', 4)
init_sizes_16, conctd_16, latency_16 = readJoinThroughput('../results/join_throughput.csv', 16)
sizes_con_4, avg_lat_list_con_4, err_list_con_4 = parse('true', init_sizes_4, conctd_4, latency_4)
sizes_ncon_4, avg_lat_list_ncon_4, err_list_ncon_4 = parse('false', init_sizes_4, conctd_4, latency_4)
sizes_con_16, avg_lat_list_con_16, err_list_con_16 = parse('true', init_sizes_16, conctd_16, latency_16)
sizes_ncon_16, avg_lat_list_ncon_16, err_list_ncon_16 = parse('false', init_sizes_16, conctd_16, latency_16)
plotJoinThroughput(sizes_con_4, sizes_ncon_4, sizes_con_16, sizes_ncon_16, avg_lat_list_con_4, avg_lat_list_ncon_4, avg_lat_list_con_16, avg_lat_list_ncon_16)

#plotJoinThroughput(sizes_con_4[:-1], sizes_ncon_4[:-1], sizes_con_16[:-1], sizes_ncon_16[:-1], avg_lat_list_con_4[:-1], avg_lat_list_ncon_4[:-1], avg_lat_list_con_16[:-1], avg_lat_list_ncon_16[:-1])


