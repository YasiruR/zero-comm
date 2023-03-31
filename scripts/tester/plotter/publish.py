#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
Created on Thu Mar 30 09:58:32 2023

@author: yasi
"""

from matplotlib import pyplot as plt
import numpy as np
import csv

def read(fileName, topic):
    init_sizes = []
    batch_sizes = []
    pings = []
    latency = []
    
    with open(fileName) as file:        
        reader = csv.reader(file)
        line_index = 0
        for row in reader:
            if line_index > 0:
                if row[0] != topic:
                    continue
                if float(row[4]) != 100.0:
                    continue
                init_sizes.append(int(row[1]))
                batch_sizes.append(int(row[2]))
                pings.append(int(row[3]))
                latency.append(float(row[5]))
            line_index += 1
    return init_sizes, batch_sizes, pings, latency

def parse(filtr_batch, init_sizes, batch_sizes, latency):
    sizes = []
    avg_lat_list = []
    err_list = []
    
    tmp_size = init_sizes[0]
    tmp_lat_list = []
    for i in range(len(latency)):
        if batch_sizes[i] != filtr_batch:
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

def plot(sizes_s, sizes_m, avg_lat_list_s, avg_lat_list_m, err_list_s, err_list_m, labels):
    fig, ax = plt.subplots()
    clrs = ["blue", "green", "brown", "purple"]
    j = 0
    for i in range(len(sizes_s)):
        ax.errorbar(sizes_s[i], avg_lat_list_s[i], yerr=err_list_s[i], ecolor="red", color=clrs[j], label=labels[j])
        j += 1
        
    for i in range(len(sizes_m)):
        ax.errorbar(sizes_m[i], avg_lat_list_m[i], yerr=err_list_m[i], ecolor="red", color=clrs[j], label=labels[j])
        j += 1
    
    ax.set_xlabel('initial group size')
    ax.set_ylabel('average time (ms)')
    ax.set_title('Latency for publish operation')
    plt.rc('grid', linestyle="--", color='#C6C6C6')
    plt.legend()
    plt.grid()
    plt.savefig('publish_latency.pdf', bbox_inches="tight")
    plt.show()


# for latency graph
init_sizes_s, batch_sizes_s, pings_s, latency_s = read('../results/publish_latency.csv', 'sq-c-o-topic')
init_sizes_m, batch_sizes_m, pings_m, latency_m = read('../results/publish_latency.csv', 'mq-c-o-topic')

sizes_s = []
avg_lat_list_s = []
err_list_s = []
for i in range(len(batch_sizes_s)):
    tmp_sizes, tmp_avg_list, tmp_err_list = parse(batch_sizes_s[i], init_sizes_s, batch_sizes_s, latency_s)
    sizes_s.append(tmp_sizes)
    avg_lat_list_s.append(tmp_avg_list)
    err_list_s.append(tmp_err_list)
    
sizes_m = []
avg_lat_list_m = []
err_list_m = []
for i in range(len(batch_sizes_m)):
    tmp_sizes, tmp_avg_list, tmp_err_list = parse(batch_sizes_m[i], init_sizes_m, batch_sizes_m, latency_m)
    sizes_m.append(tmp_sizes)
    avg_lat_list_m.append(tmp_avg_list)
    err_list_m.append(tmp_err_list)

labels = ["batch=1,single-queue", "batch=50,single-queue", "batch=100,single-queue", "batch=1,multiple-queue", "batch=50,multiple-queue", "batch=100,multiple-queue"]
plot(sizes_s, sizes_m, avg_lat_list_s, avg_lat_list_m, err_list_s, err_list_m, labels)

# for success-rate graph
def readSuccess(fileName, topic):
    init_sizes = []
    batch_sizes = []
    sucs_list = []
    
    with open(fileName) as file:        
        reader = csv.reader(file)
        line_index = 0
        for row in reader:
            if line_index > 0:
                if row[0] != topic:
                    continue
                init_sizes.append(int(row[1]))
                batch_sizes.append(int(row[2]))
                sucs_list.append(float(row[4]))
            line_index += 1
    return init_sizes, batch_sizes, sucs_list

sucs_init_sizes_s, sucs_batch_sizes_s, sucs_list_s = readSuccess('../results/publish_latency.csv', 'sq-c-o-topic')
sucs_init_sizes_m, sucs_batch_sizes_m, sucs_list_m = readSuccess('../results/publish_latency.csv', 'mq-c-o-topic')

def parseSuccess(filtr_batch, init_sizes, batch_sizes, sucs_list):
    sizes = []
    avg_sucs_list = []
    err_list = []
    
    tmp_size = init_sizes[0]
    tmp_sucs_list = []
    for i in range(len(sucs_list)):
        if batch_sizes[i] != filtr_batch:
            continue
        if tmp_size != init_sizes[i]:
            avg_sucs_list.append(np.mean(tmp_sucs_list))
            avg_sucs_list.append(np.mean())
            err_list.append(np.std(tmp_sucs_list))
            sizes.append(tmp_size)
            tmp_size = init_sizes[i]
            tmp_lat_list = [sucs_list[i]]
        else:
            tmp_lat_list.append(sucs_list[i])

    avg_sucs_list.append(np.mean(tmp_sucs_list))
    err_list.append(np.std(tmp_lat_list))
    sizes.append(tmp_size)
    return sizes, avg_sucs_list, err_list    

def plotSuccess(sizes_s, sizes_m, avg_sucs_list_s, avg_sucs_list_m, sucs_err_list_s, sucs_err_list_m, labels):
    fig, ax = plt.subplots()
    clrs = ["blue", "green", "brown", "purple"]
    j = 0
    for i in range(len(sizes_s)):
        ax.errorbar(sizes_s[i], avg_sucs_list_s[i], yerr=sucs_err_list_s[i], ecolor="red", color=clrs[j], label=labels[j])
        j += 1
        
    for i in range(len(sizes_m)):
        ax.errorbar(sizes_m[i], avg_sucs_list_m[i], yerr=sucs_err_list_m[i], ecolor="red", color=clrs[j], label=labels[j])
        j += 1
    
    ax.set_xlabel('initial group size')
    ax.set_ylabel('success rate (%)')
    ax.set_title('Success rates for publish operation')
    plt.rc('grid', linestyle="--", color='#C6C6C6')
    plt.legend()
    plt.grid()
    plt.savefig('publish_success.pdf', bbox_inches="tight")
    plt.show()

sucs_sizes_s = []
avg_sucs_list_s = []
sucs_err_list_s = []
for i in range(len(sucs_batch_sizes_s)):
    tmp_sizes, tmp_avg_list, tmp_err_list = parseSuccess(batch_sizes_s[i], sucs_init_sizes_s, sucs_batch_sizes_s, sucs_list_s)
    sucs_sizes_s.append(tmp_sizes)
    avg_sucs_list_s.append(tmp_avg_list)
    sucs_err_list_s.append(tmp_err_list)
    
sucs_sizes_m = []
avg_sucs_list_m = []
sucs_err_list_m = []
for i in range(len(sucs_batch_sizes_m)):
    tmp_sizes, tmp_avg_list, tmp_err_list = parseSuccess(batch_sizes_m[i], sucs_init_sizes_m, sucs_batch_sizes_m, sucs_list_m)
    sucs_sizes_m.append(tmp_sizes)
    avg_sucs_list_m.append(tmp_avg_list)
    sucs_err_list_m.append(tmp_err_list)

plotSuccess(sucs_sizes_s, sucs_sizes_m, avg_sucs_list_s, avg_sucs_list_m, sucs_err_list_s, sucs_err_list_m, labels)










