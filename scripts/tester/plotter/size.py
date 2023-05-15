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
    ax.errorbar(init_sizes, didcomm_sizes, marker='.', markerfacecolor='grey', markeredgecolor='black')
    
    ax.set_xlabel('original message size (bytes)')
    ax.set_ylabel('DIDComm message size (bytes)')
    ax.set_title('Message size transformation with DIDComm')
    plt.grid()
    plt.savefig('../../../docs/' + name + '.pdf', bbox_inches="tight")
    plt.show()
    
plot('msg_sizes')

grp_sizes = [2, 4, 8, 16, 32, 64]
befr_compr = [1756, 5050, 11617, 24726, 50962, 103534]
aftr_compr = [1400, 3512, 7711, 16120, 32860, 66385]

for i in range(len(befr_compr)):
    befr_compr[i] = float(befr_compr[i])/1000.0
    aftr_compr[i] = float(aftr_compr[i])/1000.0
    
def plotState(name):
    fig, ax = plt.subplots()    
    ax.errorbar(grp_sizes, befr_compr, marker='.', markerfacecolor='grey', markeredgecolor='black', label='before compression', ls='--')
    ax.errorbar(grp_sizes, aftr_compr, marker='.', markerfacecolor='grey', markeredgecolor='black', label='after compression', color='green')
    
    ax.set_xlabel('number of members')
    ax.set_ylabel('message size (kB)')
    ax.set_title('State message sizes in ZeroComm')
    plt.legend()
    plt.grid()
    plt.savefig('../../../docs/' + name + '.pdf', bbox_inches="tight")
    plt.show()
    
plotState('state_msgs')