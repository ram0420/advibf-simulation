

import time
import os

from mininet.log import info
from mininet.node import Node

from minindn.minindn import Minindn
from minindn.apps.app_manager import AppManager

from dv import NDNd_DV, DEFAULT_NETWORK

PREFIX_FILE_NAME = 'file0.txt'

def setup(ndn: Minindn, network=DEFAULT_NETWORK) -> None:
    time.sleep(1) # wait for fw to start

    NDNd_DV.init_trust()
    info('Starting ndn-dv on nodes\n')
    AppManager(ndn, ndn.net.hosts, NDNd_DV, network=network)

def converge(nodes: list[Node], deadline=30, network=DEFAULT_NETWORK) -> int:
    info('Waiting for routing to converge\n')
    start = time.time()
    while time.time() - start < deadline:
        time.sleep(1)
        if is_converged(nodes, network=network):
            total = round(time.time() - start)
            info(f'Routing converged in {total} seconds\n')
            return total

    raise Exception('Routing did not converge')


def is_converged(nodes: list[Node], network=DEFAULT_NETWORK) -> bool:
    prefix_file_path = os.path.join(os.path.dirname(__file__), PREFIX_FILE_NAME)
    with open(prefix_file_path, 'r') as f:
        lines = [line.strip() for line in f if line.strip()]

    converged = True
    for node in nodes:
        routes = node.cmd('ndnd fw route-list')
        for line in lines:
            try:
                node_name, prefix = line.split('/', 1)
                full_prefix = f'{network}/{node_name}/{prefix}'
                if full_prefix not in routes:
                    info(f'Routing not converged on {node.name} for {full_prefix}\n')
                    converged = False
                    break
            except ValueError:
                continue

        if not converged:
            return False
    return True


def converge_ibf(nodes: list[Node], deadline=30, network=DEFAULT_NETWORK) -> int:
    info('Waiting for routing to converge\n')
    start = time.time()
    while time.time() - start < deadline:
        time.sleep(1)
        if is_converged_ibf(nodes, network=network):
            total = round(time.time() - start)
            info(f'Routing converged in {total} seconds\n')
            return total

    raise Exception('Routing did not converge')

def is_converged_ibf(nodes: list[Node], network=DEFAULT_NETWORK) -> bool:
    base_dir = os.path.dirname(__file__)
    all_prefixes = []

    # 모든 prefix_{node}.txt 수집
    for node in nodes:
        prefix_file = os.path.join(base_dir, f'prefix_{node.name}.txt')
        if not os.path.exists(prefix_file):
            continue
        with open(prefix_file, 'r') as f:
            prefixes = [line.strip().lstrip('/') for line in f if line.strip()]
        for prefix in prefixes:
            full_prefix = f'{network}/{node.name}/{prefix}'
            all_prefixes.append(full_prefix)

    # 모든 노드가 모든 prefix를 라우팅 테이블에 가지고 있는지 확인
    for node in nodes:
        routes = node.cmd('ndnd fw route-list')
        for prefix in all_prefixes:
            if prefix not in routes:
                info(f'Routing not converged on {node.name} for {prefix}\n')
                return False

    return True


def converge_new_prefix(nodes: list[Node], deadline=30, network=DEFAULT_NETWORK) -> int:
    info('Waiting for NEW prefixes to converge\n')
    start = time.time()
    while time.time() - start < deadline:
        time.sleep(1)
        if is_converged_new_prefix(nodes, network=network):
            total = round(time.time() - start)
            info(f'New prefix routing converged in {total} seconds\n')
            return total

    raise Exception('New prefix routing did not converge')


def is_converged_new_prefix(nodes: list[Node], network=DEFAULT_NETWORK) -> bool:
    base_dir = os.path.dirname(__file__)
    new_prefix_file = os.path.join(base_dir, 'prefix_new.txt')
    if not os.path.exists(new_prefix_file):
        info('[WARN] prefix_new.txt not found\n')
        return False

    with open(new_prefix_file, 'r') as f:
        new_prefixes = [line.strip().lstrip('/') for line in f if line.strip()]

    for node in nodes:
        routes = node.cmd('ndnd fw route-list')
        for prefix in new_prefixes:
            full_prefix = f'{network}/a/{prefix}'
            if full_prefix not in routes:
                info(f'Routing not converged on {node.name} for {full_prefix}\n')
                return False

    return True
