# espercli

This package provides a unified command line interface to Esper API Services. The espercli package works on Python version 3.6.8.
## Installation
The easiest way to install espercli is to use `pip` in a ``virtualenv``:
```
$ pip install espercli
```
or, if you are not installing in a ``virtualenv``, to install globally:
```
$ sudo pip install espercli
```
or for your user:
```
$ pip install --user espercli
```
If you have the espercli installed and want to upgrade to the latest version
you can run:
```
$ pip install --upgrade espercli
```
This will install the espercli package as well as all dependencies.  You can also just `download the tarball`.  Once you have the
espercli directory structure on your workstation, you can just run:
```
$ cd <path_to_esper>
$ python setup.py install
```

### Getting Started

Before using espercli, you need to tell it about your Esper credentials. The way to get started is to run the espercli configure command:
```
$ espercli configure
$ Enter your Username: foo
$ Enter your Password: bar
$ Enter your Host Endpoint: develop
$ Enter your Enterprise ID: <UUID for Enterprise>
```

### Getting Help
We use GitHub issues for tracking bugs and feature requests.
