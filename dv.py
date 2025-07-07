import json
import subprocess
import shutil

from minindn.apps.application import Application

DEFAULT_NETWORK = '/minindn'

TRUST_ROOT_NAME: str = None
TRUST_ROOT_PATH = '/tmp/mn-dv-root'

class NDNd_DV(Application):
    config: str
    network: str

    def __init__(self, node, network=DEFAULT_NETWORK):
        Application.__init__(self, node)
        self.network = network

         # 1) 내 인터페이스별 URI 목록 생성
        self_uris = []
        for intf in node.intfList():
            ip = intf.IP()
            self_uris.append(f"udp4://{ip}:6363")

        if not shutil.which('ndnd'):
            raise Exception('ndnd not found in PATH, did you install it?')

        if TRUST_ROOT_NAME is None:
            raise Exception('Trust root not initialized (call NDNDV.init_trust first)')

        self.init_keys()

        config = {
            'dv': {
                # 'self_uri': f"udp4://{node.IP()}:6363",
                'self_uris':       self_uris,
                'network': network,
                'router': f"{network}/{node.name}",
                'keychain': f'dir://{self.homeDir}/dv-keys',
                'trust_anchors': [TRUST_ROOT_NAME],
                'neighbors': list(self.neighbors()),
            }
        }

        self.config = f'{self.homeDir}/dv.config.json'
        with open(self.config, 'w') as f:
            json.dump(config, f, indent=4)

    def start(self):
        Application.start(self, ['ndnd', 'dv', 'run', self.config], logfile='dv.log')

    @staticmethod
    def init_trust(network=DEFAULT_NETWORK) -> None:
        global TRUST_ROOT_NAME
        out = subprocess.check_output(f'ndnd sec keygen {network} ed25519 > {TRUST_ROOT_PATH}.key', shell=True)
        out = subprocess.check_output(f'ndnd sec sign-cert {TRUST_ROOT_PATH}.key < {TRUST_ROOT_PATH}.key > {TRUST_ROOT_PATH}.cert', shell=True)
        out = subprocess.check_output(f'cat {TRUST_ROOT_PATH}.cert | grep "Name:" | cut -d " " -f 2', shell=True)
        TRUST_ROOT_NAME = out.decode('utf-8').strip()

    def init_keys(self) -> None:
        self.node.cmd(f'rm -rf dv-keys && mkdir -p dv-keys')
        self.node.cmd(f'ndnd sec keygen {self.network}/{self.node.name}/32=DV ed25519 > dv-keys/{self.node.name}.key')
        self.node.cmd(f'ndnd sec sign-cert {TRUST_ROOT_PATH}.key < dv-keys/{self.node.name}.key > dv-keys/{self.node.name}.cert')
        self.node.cmd(f'cp {TRUST_ROOT_PATH}.cert dv-keys/')

    # def neighbors(self):
    #     for intf in self.node.intfList():
    #         other_intf = intf.link.intf2 if intf.link.intf1 == intf else intf.link.intf1
    #         # yield {"uri": f"udp4://{other_intf.IP()}:6363"}
    #         yield {
    #         "name": f"{self.network}/{other_intf.node.name}.Router",
    #         "uri": f"udp4://{other_intf.IP()}:6363"
    #     }
    
    # def neighbors(self):
    #     for intf in self.node.intfList():
    #         # 인터페이스가 연결된 상대 노드(라우터) 찾기
    #         other_intf = intf.link.intf2 if intf.link.intf1 == intf else intf.link.intf1
    #         peer = other_intf.node
    #         # peer.IP() → 라우터의 주 IP(=SelfUri에 쓴 IP)
    #         yield {
    #             "name": f"{self.network}/{peer.name}.Router",
    #             "uri":  f"udp4://{peer.IP()}:6363"
    #         }

    def neighbors(self):
        for intf in self.node.intfList():
            # 1) 이 인터페이스가 잇닿은 피어 인터페이스 찾기
            other_intf = intf.link.intf2 if intf.link.intf1 == intf else intf.link.intf1
            peer = other_intf.node
            # 2) 송신에 사용할 출발 인터페이스 URI
            from_uri = f"udp4://{intf.IP()}:6363"
            # 3) 수신 대상 URI
            to_uri   = other_intf.IP()
            yield {
                "name": f"{self.network}/{peer.name}",
                "uri":  f"udp4://{to_uri}:6363",      # ← other_intf.IP() 사용
                "from": from_uri,
           }

    # def neighbors(self):
    #     for intf in self.node.intfList():
    #         other_intf = intf.link.intf2 if intf.link.intf1 == intf else intf.link.intf1
    #         # ① 진짜 “이웃 인터페이스 IP” 를 뽑아야 합니다.
    #         nbr_ip = other_intf.IP()
    #         yield {
    #             "name": f"{self.network}/{other_intf.node.name}.Router",
    #             "uri":  f"udp4://{nbr_ip}:6363",      # ← other_intf.IP() 사용
    #         }