#!/usr/bin/env python3

from matplotlib import pyplot as plt
import csv

# add error bars
# draw all in one?

def readData(fileName, topic):
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

def plot(filtr, init_sizes, conctd, latency):
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
    
#    plt.savefig('get_line.pdf', bbox_inches="tight")
    plt.show()

topic = 'sq-c-o-topic'
init_sizes, conctd, latency = readData('../results/join.csv', topic)
plot(topic, init_sizes, conctd, latency)

