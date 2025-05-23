# Copyright 2023 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""A pipeline that passes a secret as a volume to a container."""
from kfp import dsl
from kfp import kubernetes
from kfp import compiler

# Note: this sample will only work if this secret is pre-created before running this pipeline.
# Is is pre-created by default only in the Google Cloud distribution listed here:
# https://www.kubeflow.org/docs/started/installing-kubeflow/#packaged-distributions-of-kubeflow
# If you are using a different distribution, you'll need to pre-create the secret yourself, or
# use a different secret that you know will exist.
@dsl.component
def comp():
    import os
    import sys
    username_path = os.path.join('/mnt/my_vol', "username")

    # Check if the secret exists
    if not os.path.exists(username_path):
        raise Exception('Secret not found')

    # Open the secret
    with open(username_path, 'rb') as secret_file:
        username = secret_file.read()

    # Decode the secret
    username = username.decode('utf-8')

    # Print the secret
    print(f"username: {username}")
    assert username == "user1"


@dsl.pipeline
def pipeline_secret_volume(secret_param: str = "test-secret-1"):
    task = comp()
    kubernetes.use_secret_as_volume(
        task, secret_name=secret_param, mount_path='/mnt/my_vol')


if __name__ == '__main__':
    compiler.Compiler().compile(
        pipeline_func=pipeline_secret_volume, 
        package_path='pipeline_secret_volume.yaml')