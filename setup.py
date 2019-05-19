
from setuptools import setup, find_packages

VERSION = "0.0.1"

f = open('README.md', 'r', encoding='utf-8', errors='ignore')
LONG_DESCRIPTION = f.read()
f.close()

setup(
    name='espercli',
    version=VERSION,
    description='Esper CLI tool to manage resources on Esper.io API service',
    long_description=LONG_DESCRIPTION,
    long_description_content_type='text/markdown',
    author='Esper',
    author_email='developer@esper.io',
    url='https://github.com/esper-io/esper-cli/',
    license='Apache 2.0',
    packages=find_packages(exclude=['ez_setup', 'tests*']),
    package_data={'esper': ['templates/*']},
    include_package_data=True,
    entry_points="""
        [console_scripts]
        espercli = esper.main:main
    """,
    install_requires=[
        'cement==3.0.2',
        'clint>=0.5.1',
        'colorlog>=4.0.2',
        'crayons>=0.2.0',
        'esperclient>=0.0.6',
        'jinja2>=2.10.1',
        'pyyaml>=5.1',
        'tabulate>=0.8.3',
        'tinydb>=3.13.0',
    ],
)
