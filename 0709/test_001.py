
# import random
# import os
# import time

# from mininet.log import info
# from minindn.minindn import Minindn
# from minindn.apps.app_manager import AppManager
# from minindn.apps.nfd import Nfd

# from fw import NDNd_FW
# import dv_util

# PREFIX_FILE_NAME = 'prefix_6*7.txt'

# def scenario_ndnd_fw(ndn: Minindn):
#     scenario(ndn, fw=NDNd_FW)

# def scenario_nfd(ndn: Minindn):
#     scenario(ndn, fw=Nfd)

# def scenario(ndn: Minindn, fw=None, network='/minindn'):
#     """
#     Simple file transfer scenario with NDNd and NFD forwarders.
#     This tests routing convergence and cat/put operations.
#     Also tests routing compatibility for both NDNd and NFD.
#     """

#     info('Starting forwarder on nodes\n')
#     AppManager(ndn, ndn.net.hosts, fw)

#     dv_util.setup(ndn, network=network)

#     # prefix 파일
#     info('Reading prefixes from{PREFIX_FILE_NAME}\n')
#     prefix_file_path = os.path.join(os.path.dirname(__file__), PREFIX_FILE_NAME)

#     # 파일의 각 줄을 읽어 리스트로 저장 (빈 줄 제거 포함)
#     with open(prefix_file_path, 'r') as f:
#         lines = [line.strip() for line in f if line.strip()]

#     # put에 사용할 임시 랜덤 파일 생성 (10MB)
#     test_file = '/tmp/test.bin'
#     os.system(f'dd if=/dev/urandom of={test_file} bs=10M count=1')

#     # 파일에서 읽은 각 prefix에 대해 put 실행
#     for line in lines:
#         try:
#             # 'node1/prefixA' → node_name='node1', prefix='prefixA'
#             node_name, prefix = line.split('/', 1)
#         except ValueError:
#             info(f'Invalid line format in {PREFIX_FILE_NAME}: {line}\n')
#             continue  # 잘못된 형식이면 스킵

#         # node_name에 해당하는 실제 노드 객체 찾기
#         node = next((n for n in ndn.net.hosts if n.name == node_name), None)
#         if not node:
#             info(f'Node {node_name} not found in topology\n')
#             continue  # 해당 노드가 없으면 스킵

#         # 전체 prefix 경로 구성 (예: /minindn/node1/prefixA)
#         full_prefix = f'{network}/{node_name}/{prefix}'
        
#         # ndnd put 명령어 생성 및 실행 (백그라운드 실행)
#         cmd = f'ndnd put --expose "{full_prefix}" < {test_file} &'
#         info(f'{node.name} {cmd}\n')
#         node.cmd(cmd)
#         time.sleep(0.5)

#     # 모든 put 작업이 안정화되도록 대기
#     info('Waiting for all put operations to settle\n')
#     time.sleep(3)
#     info('All put operations completed.\n')

#     # convergence 확인
#     info('running routing convergence...\n')
#     dv_util.converge(ndn.net.hosts, network=network)
#     info('running routing convergence completed. \n')

#     check_missing_prefixes.check_missing_prefixes(ndn, prefix_file=prefix_file_path)



#     #new prefix 생성

#     info('Put operations completed\n')


import random
import os
import time

from mininet.log import info
from minindn.minindn import Minindn
from minindn.apps.app_manager import AppManager
from minindn.apps.nfd import Nfd

from fw import NDNd_FW
import dv_util

PREFIX_FILE_NAME = 'file0.txt'

def scenario_ndnd_fw(ndn: Minindn):
    scenario(ndn, fw=NDNd_FW)

def scenario_nfd(ndn: Minindn):
    scenario(ndn, fw=Nfd)

def scenario(ndn: Minindn, fw=None, network='/minindn'):
    """
    Simple file transfer scenario with NDNd and NFD forwarders.
    This tests routing convergence and cat/put operations.
    Also tests routing compatibility for both NDNd and NFD.
    """

    info('Starting forwarder on nodes\n')
    AppManager(ndn, ndn.net.hosts, fw)

    dv_util.setup(ndn, network=network)

    # prefix 파일
    info('Reading prefixes from{PREFIX_FILE_NAME}\n')
    prefix_file_path = os.path.join(os.path.dirname(__file__), PREFIX_FILE_NAME)

    # 파일의 각 줄을 읽어 리스트로 저장 (빈 줄 제거 포함)
    with open(prefix_file_path, 'r') as f:
        lines = [line.strip() for line in f if line.strip()]

    # put에 사용할 임시 랜덤 파일 생성 (10MB)
    test_file = '/tmp/test.bin'
    os.system(f'dd if=/dev/urandom of={test_file} bs=10M count=1')

    # 파일에서 읽은 각 prefix에 대해 put 실행
    for line in lines:
        try:
            # 'node1/prefixA' → node_name='node1', prefix='prefixA'
            node_name, prefix = line.split('/', 1)
        except ValueError:
            info(f'Invalid line format in {PREFIX_FILE_NAME}: {line}\n')
            continue  # 잘못된 형식이면 스킵

        # node_name에 해당하는 실제 노드 객체 찾기
        node = next((n for n in ndn.net.hosts if n.name == node_name), None)
        if not node:
            info(f'Node {node_name} not found in topology\n')
            continue  # 해당 노드가 없으면 스킵

        # 전체 prefix 경로 구성 (예: /minindn/node1/prefixA)
        full_prefix = f'{network}/{node_name}/{prefix}'
        
        # ndnd put 명령어 생성 및 실행 (백그라운드 실행)
        cmd = f'ndnd put --expose "{full_prefix}" < {test_file} &'
        info(f'{node.name} {cmd}\n')
        node.cmd(cmd)
        time.sleep(0.5)

    # 모든 put 작업이 안정화되도록 대기
    info('Waiting for all put operations to settle\n')
    time.sleep(3)
    info('All put operations completed.\n')

    # convergence 확인
    info('running routing convergence...\n')
    dv_util.converge(ndn.net.hosts, network=network)
    info('running routing convergence completed. \n')

    #new prefix 생성

    info('Put operations completed\n')