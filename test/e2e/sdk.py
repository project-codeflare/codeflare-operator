from codeflare_sdk.cluster.cluster import Cluster, ClusterConfiguration
# from codeflare_sdk.cluster.auth import TokenAuthentication
from codeflare_sdk.job.jobs import DDPJobDefinition

cluster = Cluster(ClusterConfiguration(
    name='mnist',
    # namespace='default',
    min_worker=1,
    max_worker=1,
    min_cpus=0.2,
    max_cpus=1,
    min_memory=0.5,
    max_memory=1,
    gpu=0,
    instascale=False,
))

cluster.up()

cluster.status()

cluster.wait_ready()

cluster.status()

cluster.details()

jobdef = DDPJobDefinition(
    name="mnist",
    script="/test/job/mnist.py",
    scheduler_args={"requirements": "/test/runtime/requirements.txt"}
)
job = jobdef.submit(cluster)

job.status()

print(job.logs())

cluster.down()
