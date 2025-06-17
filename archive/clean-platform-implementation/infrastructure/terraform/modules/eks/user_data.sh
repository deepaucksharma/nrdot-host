#!/bin/bash
set -o xtrace

# Configure EKS
/etc/eks/bootstrap.sh ${cluster_name} \
  --b64-cluster-ca ${cluster_ca} \
  --apiserver-endpoint ${cluster_endpoint}

# Additional user data
${additional_userdata}