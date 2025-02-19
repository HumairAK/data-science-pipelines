import setuptools
import os
import re

NAME = 'kfp-common'
VERSION = '0.0.0'
setuptools.setup(
    name=NAME,
    version=VERSION,
    description='Kubeflow Pipelines common spec',
    author='kfp-authors',
    author_email='kubeflow-pipelines@google.com',
    url='https://github.com/kubeflow/pipelines',
    packages=setuptools.find_namespace_packages(include=['kfp.*']),
    python_requires='>=3.9.0',
    install_requires=['protobuf>=4.21.1,<5'],
    include_package_data=True,
    license='Apache 2.0',
)