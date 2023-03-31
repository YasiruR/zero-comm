#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
Created on Thu Mar 30 09:58:32 2023

@author: yasi
"""

def readPublishLatency(fileName, topic):
    init_sizes = []
    conctd = []
    latency = []
    with open(fileName) as file:        
        reader = csv.reader(file)
        line_index = 0
        for row in reader:
            if line_index > 0:
                if int(row[1]) == 64:
                    continue
                
                init_sizes.append(int(row[1]))
                conctd.append(row[5])
                latency.append(float(row[6]))
            line_index += 1
    return init_sizes, conctd, latency