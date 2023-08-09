import sys

from time import sleep

from torchx.specs.api import AppState, is_terminal

from codeflare_sdk.cluster.cluster import Cluster, ClusterConfiguration
from codeflare_sdk.job.jobs import DDPJobDefinition

namespace = sys.argv[1]

cluster = Cluster(ClusterConfiguration(
    name='mnist',
    namespace=namespace,
    num_workers=1,
    min_cpus='500m',
    max_cpus=1,
    min_memory=0.5,
    max_memory=1,
    num_gpus=0,
    instascale=False,
))

cluster.up()

cluster.status()

cluster.wait_ready()

cluster.status()

cluster.details()

jobdef = DDPJobDefinition(
    name="mnist",
    script="mnist.py",
    scheduler_args={"requirements": "requirements.txt"},
)
job = jobdef.submit(cluster)

done = False
time = 0
timeout = 300
while not done:
    status = job.status()
    if is_terminal(status.state):
        break
    if not done:
        print(status)
        if timeout and time >= timeout:
            raise TimeoutError(f"job has timed out after waiting {timeout}s")
        sleep(5)
        time += 5

print(f"Job has completed: {status.state}")

print(job.logs())

cluster.down()

if not status.state == AppState.SUCCEEDED:
    exit(1)
else:
    exit(0)
