# Build the manager binary
FROM pytorch/pytorch:1.11.0-cuda11.3-cudnn8-runtime

WORKDIR /test
COPY entrypoint.sh entrypoint.sh

# Install MNIST requirements
COPY mnist_pip_requirements.txt requirements.txt
RUN pip install --requirement requirements.txt

# Prepare MNIST script
COPY mnist.py mnist.py
COPY download_dataset.py download_dataset.py
RUN torchrun download_dataset.py

USER 65532:65532
WORKDIR /workdir
ENTRYPOINT ["/test/entrypoint.sh"]
