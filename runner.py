import random
import os
import time

from types import FunctionType

from mininet.log import setLogLevel, info
from minindn.minindn import Minindn

import test_001
import test_002
import test_ibf

def run(scenario: FunctionType, **kwargs) -> None:
    try:
        random.seed(0)

        info(f"===================================================\n")
        start = time.time()
        scenario(ndn, **kwargs)
        info(f'Scenario completed in: {time.time()-start:.2f}s\n')
        info(f"===================================================\n\n")

        # Call all cleanups without stopping the network
        # This ensures we don't recreate the network for each test
        for cleanup in reversed(ndn.cleanups):
            cleanup()
    except Exception as e:
        ndn.stop()
        raise e
    finally:
        # kill everything we started just in case ...
        os.system('pkill -9 ndnd')
        os.system('pkill -9 nfd')

if __name__ == '__main__':
    setLogLevel('info')

    #  로그 파일 초기화 (여기 넣기!)
    if os.path.exists("/tmp/final_advert_log.txt"):
        os.remove("/tmp/final_advert_log.txt")


    Minindn.cleanUp()
    Minindn.verifyDependencies()

    ndn = Minindn()
    ndn.start()

    # run(test_001.scenario_ndnd_fw)


    run(test_ibf.scenario_ndnd_fw)

    #run(test_001.scenario_nfd)
    #run(test_002.scenario)

    ndn.stop()