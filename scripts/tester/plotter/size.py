#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
Created on Sun Apr 30 10:58:17 2023

@author: yasi
"""

from matplotlib import pyplot as plt

init_sizes = [1, 5, 27, 59, 135, 564, 1371, 2788]
didcomm_sizes = [681, 685, 713, 757, 857, 1429, 2505, 4397]

def plot(name):
    fig, ax = plt.subplots()    
    ax.errorbar(init_sizes, didcomm_sizes, marker='.', markerfacecolor='red', markeredgecolor='black')
    
    ax.set_xlabel('initial size (bytes)')
    ax.set_ylabel('DIDComm message size (bytes)')
    ax.set_title('Message size transformation with DIDComm')
    plt.grid()
    plt.savefig('../../../docs/' + name + '.pdf', bbox_inches="tight")
    plt.show()
    
plot('msg_sizes')


def diff():
    for i in range(len(init_sizes)):
        d = didcomm_sizes[i]-init_sizes[i]
        print(d, init_sizes[i])
        
diff()