
from setuptools import setup, find_packages


VERSION = "0.0.10"

f = open('README.md', 'r', encoding='utf-8', errors='ignore')
LONG_DESCRIPTION = f.read()
f.close()

setup(
    name='espercli',
    version=VERSION,
    description='Esper CLI is Command line tool for the Esper APIs',
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
        'esperclient>=0.0.10',
        'jinja2>=2.10.1',
        'pyyaml>=5.1',
        'requests==2.22.0',
        'tabulate>=0.8.3',
        'tinydb>=3.13.0',
        'tqdm>=4.32.1',
        'pyOpenSSL==19.0.0'
    ],
)
