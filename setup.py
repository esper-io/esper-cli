
from setuptools import setup, find_packages
from esper.core.version import get_version

VERSION = get_version()

f = open('README.md', 'r')
LONG_DESCRIPTION = f.read()
f.close()

setup(
    name='esper',
    version=VERSION,
    description='Esper CLI tool to manage resources on Esper.io API service',
    long_description=LONG_DESCRIPTION,
    long_description_content_type='text/markdown',
    author='Jeryn Mathew ',
    author_email='jeryn@shoonya.io',
    url='https://github.com/esper-io/esper-cli/',
    license='Apache 2.0',
    packages=find_packages(exclude=['ez_setup', 'tests*']),
    package_data={'esper': ['templates/*']},
    include_package_data=True,
    entry_points="""
        [console_scripts]
        esper = esper.main:main
    """,
)
