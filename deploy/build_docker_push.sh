#!/bin/bash
passwd=$1
version=$2
echo {passwd} | docker login --username=qinguoyi --password-stdin
docker build -t object-storage-proxy:${version} -f Dockerfile  .
docker tag object-storage-proxy:${version} qinguoyi/object-storage-proxy:${version}
docker push qinguoyi/object-storage-proxy:${version}
if [ $? -eq 0 ]; then
 echo "push success"
else
 echo "push failed"
fi