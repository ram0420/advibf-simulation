import os
import time

from mininet.log import info
from minindn.minindn import Minindn
from minindn.apps.app_manager import AppManager
from minindn.apps.nfd import Nfd

from fw import NDNd_FW
import dv_util

PREFIX_DIR = os.path.dirname(__file__)  # prefix 파일이 있는 디렉토리
NETWORK_PREFIX = '/minindn'

def scenario_ndnd_fw(ndn: Minindn):
    scenario(ndn, fw=NDNd_FW)

def scenario_nfd(ndn: Minindn):
    scenario(ndn, fw=Nfd)

def scenario(ndn: Minindn, fw=None, network=NETWORK_PREFIX):
    info('Starting forwarder on nodes\n')
    AppManager(ndn, ndn.net.hosts, fw)

    # Distance Vector 초기화
    dv_util.setup(ndn, network=network)

    # put에 사용할 테스트 파일 생성
    test_file = '/tmp/test.bin'
    if not os.path.exists(test_file):
        os.system(f'dd if=/dev/urandom of={test_file} bs=10M count=1')

    ###############  1. 각 라우터 별로 prefixex put  #################
    router_names = [node.name for node in ndn.net.hosts]

    for router in router_names:
        prefix_file = os.path.join(PREFIX_DIR, f'prefix_{router}.txt')

        if not os.path.exists(prefix_file):
            info(f'[WARN] Prefix file not found: {prefix_file}\n')
            continue

        node = next((n for n in ndn.net.hosts if n.name == router), None)
        if not node:
            info(f'[WARN] Node not found: {router}\n')
            continue

        with open(prefix_file, 'r') as f:
            prefixes = [line.strip() for line in f if line.strip()]

        for prefix in prefixes:
            clean_prefix = prefix.lstrip('/')  
            full_prefix = f'{network}/{router}/{clean_prefix}'
            cmd = f'ndnd put --expose "{full_prefix}" < {test_file} &'
            info(f'{router} {cmd}\n')
            node.cmd(cmd)
            time.sleep(0.5)


    # 안정화를 위한 대기
    info('Waiting for all put operations to settle\n')
    time.sleep(5)

    # 수렴 실행
    info('Running routing convergence...\n')
    dv_util.converge_ibf(ndn.net.hosts, network=network)
    info('Routing convergence completed.\n')

    #/tmp/final_advert_log.txt
    write_advert_logs_to_final(ndn, "1차 수렴")

    ###############  2. prefix_new.txt 내용을 노드 a에서 put #################
    new_prefix_file = os.path.join(PREFIX_DIR, 'prefix_new.txt')
    node_a = next((n for n in ndn.net.hosts if n.name == 'a'), None)

    if node_a and os.path.exists(new_prefix_file):
        info('Putting new prefixes from prefix_new.txt via node a\n')
        with open(new_prefix_file, 'r') as f:
            new_prefixes = [line.strip() for line in f if line.strip()]

        for prefix in new_prefixes:
            clean_prefix = prefix.lstrip('/')
            full_prefix = f'{network}/a/{clean_prefix}'
            cmd = f'ndnd put --expose "{full_prefix}" < {test_file} &'
            info(f'a {cmd}\n')
            node_a.cmd(cmd)
            time.sleep(0.5)

        info('Waiting for new put operations to settle\n')
        time.sleep(3)

        # 다시 수렴 확인
        info('Running second routing convergence...\n')
        dv_util.converge_new_prefix(ndn.net.hosts, network=network)
        info('Second routing convergence completed.\n')
        
        #/tmp/final_advert_log.txt
        write_advert_logs_to_final(ndn, "2차 수렴")

    else:
        info('[WARN] prefix_new.txt not found or node a not found. Skipping second phase.\n')


    info('All put and convergence steps completed.\n')

def write_advert_logs_to_final(ndn, phase_title, output_file="/tmp/final_advert_log.txt"):
    with open(output_file, "a") as out:
        out.write(f"\n===== {phase_title} =====\n")
        for node in ndn.net.hosts:
            node_log_path = f"/tmp/minindn/{node.name}/advert_log.txt"
            out.write(f"\n--- Node {node.name} ---\n")
            if os.path.exists(node_log_path):
                with open(node_log_path, "r") as node_log:
                    out.write(node_log.read())
            else:
                out.write("[No advert log found]\n")
