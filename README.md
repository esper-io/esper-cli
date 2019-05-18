# Esper CLI
[![Build Status](https://travis-ci.com/esper-io/esper-cli.svg?branch=master)](https://travis-ci.com/esper-io/esper-cli) [![Gitter](https://badges.gitter.im/esper-dev/esper-cli.svg)](https://gitter.im/esper-dev/esper-cli?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge)

This package provides a unified command line interface to the Esper API Services.

Current stable release versions are

    API version: 1.0.0
    SDK version: 0.0.6
    CLI version: 0.0.1

## Requirements

1. **Python:** We recommend you use Python 3.6 or above.
2. **An Esper Dev Account:** You need a free Esper Dev Trial account to create an environment and generate an Esper `SERVER URL`to talk to APIs. You will choose the environment name that will then be assigned as your custom URL and when you complete the sign up process your private environment will be created. For example if your you choose the environment name of “foo” then your `SERVER URL` will be `https://foo.esper.cloud/api`. See [Requesting an Esper Dev Trial account](https://docs.esper.io/home/gettingstarted.html#setup). 
3. **Generate an API key:** API key authentication is used for accessing APIs. You will have to generate this from the Esper Dev Console once you have set up your account. The Esper Dev Console for your account can be accessed at `https://foo.espercloud.com`. See [Generating an API Key](https://docs.esper.io/home/module/genapikey.html)

## Installation

#### Using `pip install`

From PyPI
```sh
pip install espercli
```

or

From [Github](https://github.com/esper-io/esper-cli)
```sh
pip install git+https://github.com/esper-io/esper-cli.git
```

#### From source

Download/Clone the project and install via [Setuptools](http://pypi.python.org/pypi/setuptools).

```sh
git clone https://github.com/esper-io/esper-cli.git

cd esper-cli

python setup.py install
```

> You need not install setuptools separately, they are packaged along with downloaded library


### Usage

Before using espercli, you need to tell it about your Esper credentials. For this you will need `API KEY` and `HOST ENVIRONMENT` as generated in [Requirements](#requirements) section.
The way to get started is to run the espercli configure command:
```sh
$ espercli configure
$ Esper API Key: LpDriKp7MWJiRGcwc8xzREeUj8OEFa
$ Tenant name: myapp
```
To list available commands, either run `espercli` with no parameters or execute `espercli --help`:
```sh
usage: espercli [-h] [-d] [-q] [-v]
                {group-command,group,enterprise,status,install,version,device-command,app,device,configure}
                ...

Esper CLI tool to manage resources on Esper.io API service

optional arguments:
  -h, --help            show this help message and exit
  -d, --debug           full application debug mode
  -q, --quiet           suppress all console output
  -v, --version         show program's version number and exit

sub-commands:
  {group-command,group,enterprise,status,install,version,device-command,app,device,configure}
    group-command       group-command controller
    group               group controller
    enterprise          enterprise controller
    status              status controller
    install             install controller
    version             version controller
    device-command      device-command controller
    app                 app controller
    device              device controller
    configure           Configure the credentials for `esper.io` API Service

Usage: espercli <sub-command> [--options]
```

## *Commands*
### **Configure**
Configure command is used to set and modify Esper credential details and can show credential details if not given `-s` or `--set` option.
 ```sh
$ espercli configure [OPTIONS]
```

##### Options
| Name, shorthand| Default| Description|
| -------------  |:------:|:----------|
| --set, -s      |        | Set or modify credentials |
| --json, -j     |        | Render result in JSON format |

##### Example
```sh
$ espercli configure

TITLE    DETAILS
api_key  LpDriKp7MWJiRGcwc8xzREeUj8OEFa
tenant   myapp
```

### **Enterprise**
Enterprise command used to show and modify enterprise information.
```sh
$ espercli enterprise [SUB-COMMANDS]
```
#### Sub commands
#### 1. show
Show enterprise information.
```sh
$ espercli enterprise show [OPTIONS]
```
##### Options
| Name, shorthand| Default| Description|
| ------------- |:-------------:|:-----|
| --json, -j     |  | Render result in JSON format |

##### Example
```sh
$ espercli enterprise show

TITLE            DETAILS
id               595a6107-b137-448d-b217-e20cc58ee84d
name             Myapp Enterprise
display_name     Myapp
registered_name  Myapp Enterprise
address          #123, Industrial Layout, Random Avenue
location         Santa Clara, CA
zipcode          12345
email            contact@myapp.io
contact_person   Muneer
contact_number   +145678901234

$ espercli enterprise show -j
{"id": "595a6107-b137-448d-b217-e20cc58ee84d", "name": "Myapp Enterprise", "display_name": "Myapp", "registered_name": "Myapp Enterprise", "address": "#123, Industrial Layout, Random Avenue", "location": "Santa Clara, CA", "zipcode": "12345", "email": "contact@myapp.io", "contact_person": "Muneer", "contact_number": "+141234501234"}%
```

#### 2. update
Modify the enterprise information.
```sh
$ espercli enterprise update [OPTIONS]
```
##### Options
| Name, shorthand | Default| Description|
| -------------   |:------:|:----------|
| --name, -n      |        | Enterprise name |
| --dispname, -dn |        | Enterprise display name |
| --regname, -rn  |        | Enterprise registered name |
| --address, -a   |        | Enterprise address |
| --location, -l  |        | Enterprise location |
| --zipcode, -z   |        | Enterprise zip code |
| --email, -e     |        | Enterprise contact email |
| --person, -p    |        | Enterprise contact person name |
| --number, -cn   |        | Enterprise contact number |
| --json, -j      |        | Render result in JSON format |

##### Example
```sh
$ espercli enterprise update -p 'Muneer M'

TITLE            DETAILS
id               595a6107-b137-448d-b217-e20cc58ee84d
name             Myapp Enterprise
display_name     Myapp
registered_name  Myapp Enterprise
address          #123, Industrial Layout, Random Avenue
location         Santa Clara, CA
zipcode          12345
email            contact@myapp.io
contact_person   Muneer M
contact_number   +145678901234
```

### **Device**
Device used to list and show device information and set device as active for further commands.
```sh
$ espercli device [SUB-COMMANDS]
```
#### Sub commands
#### 1. list
List sub command used to list all devices and can filter results by using different options listed below. Pagination used to limit the number of results, default is 20 results per page.
```sh
$ espercli device list [OPTIONS]
```
##### Options
| Name, shorthand | Default| Description|
| -------------   |:------:|:----------|
| --limit, -l     |20      | Number of results to return per page |
| --offset, -i    |0       | The initial index from which to return the results |
| --state, -s     |        | Filter by device state, choices are [active, inactive, disabled] |
| --name, -n      |        | Filter by device name |
| --group, -g     |        | Filter by group name |
| --imei, -im     |        | Filter by device IMEI number |
| --brand, -b     |        | Filter by device brand name |
| --gms, -gm      |        | Filter by GMS and non GMS flag, choices are [true, false] |
| --json, -j      |        | Render result in JSON format |

##### Example
```sh
$ espercli device list -gm false
Number of Devices: 10

ID                                    NAME          MODEL     CURRENT STATE
62d42cff-6979-48ed-bedf-8b25052a74d0  SNA-SNL-FZH5  QUALCOMM  INACTIVE
9877c1f0-0435-4185-a41b-e896e33bd438  SNA-SNL-V84B  QUALCOMM  INACTIVE
1bab8bf7-4b12-426e-a35b-00a718ec3490  SNA-SNL-XA05  POSBANK   INACTIVE
9cdb45ed-5bc7-433a-b08b-1c0cffffebec  SNA-SNL-N7XY  Esper     DISABLED
d89a88f3-de5c-4acc-9eae-0868bd2fad15  SNA-SNL-U1K1  EMDOOR    INACTIVE
fc3af4e3-79f4-483f-986e-3af60bb58809  SNA-SNL-T1PX  Vertex    DISABLED
e2a7d069-b536-4700-b07b-4db9d9d9236c  SNA-SNL-B424  Esper     INACTIVE
218b37c5-b1cf-4768-8340-b2bc5f701b54  SNA-SNL-BGD3  EMDOOR    INACTIVE
647ef365-0b68-4fbd-aa11-febe54d668b1  SNA-SNL-8QJG  Intel     INACTIVE
c7c0382e-b911-451a-9d62-54936622d3b3  SNA-SNL-R123  QUALCOMM  DISABLED
```

#### 2. show
Show device details and set device as active. Here, device name is required to show device information.
```sh
$ espercli device show [OPTIONS] [device-name]
```
##### Options
| Name, shorthand | Default| Description|
| -------------   |:------:|:----------|
| --active, -a    |        | Set device as active for further device specific commands |
| --json, -j      |        | Render result in JSON format |

##### Example
```sh
$ espercli device show -a SNA-SNL-FZH5

TITLE          DETAILS
id             62d42cff-6979-48ed-bedf-8b25052a74d0
device_name    SNA-SNL-FZH5
suid           tFxCx1wRMnMIk7kO3GpkgX--VQEI_FQxC13D1Bh4yRA
api_level      28
template_name  NonGMS
is_gms         False
state          INACTIVE
```

#### 3. active
Active sub command used to set and reset active device and show active device information with no options.
```sh
$ espercli device active [OPTIONS]
```
##### Options
| Name, shorthand | Default| Description|
| -------------   |:------:|:----------|
| --name, -n      |        | Device name |
| --reset, -r     |        | Reset the active device |
| --json, -j      |        | Render result in JSON format |

##### Example
```sh
$ espercli device active -n SNA-SNL-FZH5

TITLE          DETAILS
id             62d42cff-6979-48ed-bedf-8b25052a74d0
device_name    SNA-SNL-FZH5
suid           tFxCx1wRMnMIk7kO3GpkgX--VQEI_FQxC13D1Bh4yRA
api_level      28
template_name  NonGMS
is_gms         False
state          INACTIVE
```

### **Group**
Group used to manage a group like list, show, create and update. Also can list devices in a group, add devices to group, remove devices and set group as active for further commands.
```sh
$ espercli group [SUB-COMMANDS]
```
#### Sub commands
#### 1. list
List sub command used to list all groups and can filter results by using different options listed below. Pagination used to limit the number of results, default is 20 results per page.
```sh
$ espercli group list [OPTIONS]
```
##### Options
| Name, shorthand | Default| Description|
| -------------   |:------:|:----------|
| --limit, -l     |20      | Number of results to return per page |
| --offset, -i    |0       | The initial index from which to return the results |
| --name, -n      |        | Filter by group name |
| --json, -j      |        | Render result in JSON format |

##### Example
```sh
$ espercli group list -n 5G
Number of Groups: 1

ID                                    NAME      DEVICE COUNT
2e5efca2-7776-442e-a5ef-c2758d4a45a3  5G                   2
```

#### 2. show
Show group details and set group as active. Here, group name is required to show group information.
```sh
$ espercli group show [OPTIONS] [group-name]
```
##### Options
| Name, shorthand | Default| Description|
| -------------   |:------:|:----------|
| --active, -a    |        | Set device as active for further device specific commands |
| --json, -j      |        | Render result in JSON format |

##### Example
```sh
$ espercli group show -a 5G

TITLE         DETAILS
id            2e5efca2-7776-442e-a5ef-c2758d4a45a3
name          5G
device_count  2
```

#### 3. active
Active sub command used to set and reset active group and show active group information with no options.
```sh
$ espercli group active [OPTIONS]
```
##### Options
| Name, shorthand | Default| Description|
| -------------   |:------:|:----------|
| --name, -n      |        | Group name |
| --reset, -r     |        | Reset the active group |
| --json, -j      |        | Render result in JSON format |

##### Example
```sh
$ espercli group active -n 5G

TITLE         DETAILS
id            2e5efca2-7776-442e-a5ef-c2758d4a45a3
name          5G
device_count  2
```

#### 4. create
Create new group.
```sh
$ espercli group create [OPTIONS]
```
##### Options
| Name, shorthand | Default| Description|
| -------------   |:------:|:----------|
| --name, -n      |        | Group name |
| --json, -j      |        | Render result in JSON format |

##### Example
```sh
$ espercli group create -n 5G

TITLE         DETAILS
id            2e5efca2-7776-442e-a5ef-c2758d4a45a3
name          5G
device_count  0
```

#### 5. update
Modify group information.
```sh
$ espercli group update [OPTIONS] [group-name]
```
##### Options
| Name, shorthand | Default| Description|
| -------------   |:------:|:----------|
| --name, -n      |        | Group new name |
| --json, -j      |        | Render result in JSON format |

##### Example
```sh
$ espercli group update -n 4G 5G

TITLE         DETAILS
id            2e5efca2-7776-442e-a5ef-c2758d4a45a3
name          4G
device_count  2
```

#### 6. delete
Remove particular group.
```sh
$ espercli group delete [group-name]
```

##### Example
```sh
$ espercli group delete 5G
Group with name 5G deleted successfully
```

#### 7. add
Add devices into a group, active group is used to add devices if `--group` or `-g` option is not given explicitly.
```sh
$ espercli group add [OPTIONS]
```
##### Options
| Name, shorthand | Default| Description|
| -------------   |:------:|:----------|
| --group, -g     |        | Group name |
| --devices, -d   |        | List of device names, list format is space separated |
| --json, -j      |        | Render result in JSON format |

##### Example
```sh
$ espercli group add -g 5G -d SNA-SNL-73YE SNA-SNL-NYWL 

TITLE         DETAILS
id            2e5efca2-7776-442e-a5ef-c2758d4a45a3
name          5G
device_count  2
```

#### 8. remove
Remove devices from a group, active group is used to add devices if `--group` or `-g` option is not given explicitly.
```sh
$ espercli group remove [OPTIONS]
```
##### Options
| Name, shorthand | Default| Description|
| -------------   |:------:|:----------|
| --group, -g     |        | Group name |
| --devices, -d   |        | List of device names, list format is space separated |
| --json, -j      |        | Render result in JSON format |

##### Example
```sh
$ espercli group remove -g 5G -d SNA-SNL-73YE SNA-SNL-NYWL 

TITLE         DETAILS
id            2e5efca2-7776-442e-a5ef-c2758d4a45a3
name          5G
device_count  0
```

#### 9. devices
List devices in a particular group, active group is used to add devices if `--group` or `-g` option is not given explicitly. Pagination used to limit the number of results, default is 20 results per page.
```sh
$ espercli group devices [OPTIONS] [group-name]
```
##### Options
| Name, shorthand | Default| Description|
| -------------   |:------:|:----------|
| --limit, -l     |20      | Number of results to return per page |
| --offset, -i    |0       | The initial index from which to return the results |
| --group, -g     |        | Group name |
| --json, -j      |        | Render result in JSON format |

##### Example
```sh
$ espercli group devices -g 5G
Number of Devices: 2

ID                                    NAME          MODEL     CURRENT STATE
3ebc3afd-249b-4f10-8561-fa1a9ddb1bb7  SNA-SNL-KX37  Shoonya   ACTIVE
c8efa083-f325-4e3b-8d20-71b7a2927ffb  SNA-SNL-3606  QUALCOMM  INACTIVE
```

### **Application**
Application command used to list, show, upload and delete applications and set application as active for further commands.
```sh
$ espercli app [SUB-COMMANDS]
```
#### Sub commands
#### 1. list
List all applications and can filter results by using different options listed below. Pagination used to limit the number of results, default is 20 results per page.
```sh
$ espercli app list [OPTIONS]
```
##### Options
| Name, shorthand | Default| Description|
| -------------   |:------:|:----------|
| --limit, -l     |20      | Number of results to return per page |
| --offset, -i    |0       | The initial index from which to return the results |
| --name, -n      |        | Filter by application name |
| --package, -p   |        | Filter by package name |
| --json, -j      |        | Render result in JSON format |

##### Example
```sh
$ espercli app list -l 5
Total Number of Applications: 76

ID                                    NAME                PACKAGE NAME                    NO. OF VERSIONS
d7131f72-17e4-40e9-bb9e-28f1fad1f623  ATID Reader         com.atid.app.atx                              1
0c067884-8d72-41b5-9ed7-3e6f1f62d99d  Call Blocker        com.sappalodapps.callblocker                  1
630dbfab-7d85-4f81-9f3b-ffb038b0df72  Root Checker Basic  com.joeykrim.rootcheck                        1
4baf7157-9fee-4dc5-ab3a-81dc983d7332  Castle Clash        com.igg.castleclash                           1
09368a1b-a9cd-45bc-8824-7190bc0f6b7e  WiFiAnalyzer        com.vrem.wifianalyzer                         1
```

#### 2. show
Show application information and set application as active. Here, application id (UUID) is required to show application information.
```sh
$ espercli app show [OPTIONS] [application-id]
```
##### Options
| Name, shorthand | Default| Description|
| -------------   |:------:|:----------|
| --active, -a    |        | Set application as active for further application specific commands |
| --json, -j      |        | Render result in JSON format |

##### Example
```sh
$ espercli app show -a 630dbfab-7d85-4f81-9f3b-ffb038b0df72

TITLE             DETAILS
id                630dbfab-7d85-4f81-9f3b-ffb038b0df72
application_name  Root Checker Basic
package_name      com.joeykrim.rootcheck
developer
category
content_rating    0.0
compatibility
version_count     1
```

#### 3. upload
Upload sub command used to upload application file. Here, application file path is required to upload file.
```sh
$ espercli app upload [OPTIONS] [application-file]
```
##### Options
| Name, shorthand | Default| Description|
| -------------   |:------:|:----------|
| --json, -j      |        | Render result in JSON format |

##### Example
```sh
$ espercli app upload ~/myapp/com.joeykrim.rootcheck-v1.1.apk

TITLE             DETAILS
id                630dbfab-7d85-4f81-9f3b-ffb038b0df72
application_name  Root Checker Basic
package_name      com.joeykrim.rootcheck
developer
category
content_rating    0.0
compatibility
version_id        e933366b-9bb2-4c41-87fe-023f839dc367
version_code      1.0
build_number      1
```

#### 4. delete
Delete sub command used to delete application. Here, application id (UUID) is required to delete application and reset active application if it is set as active.
```sh
$ espercli app delete [application-id]
```

##### Example
```sh
$ espercli app delete 630dbfab-7d85-4f81-9f3b-ffb038b0df72
Application with id 630dbfab-7d85-4f81-9f3b-ffb038b0df72 deleted successfully
```

#### 5. active
Active sub command used to set and reset active application and show active application information with no options.
```sh
$ espercli app active [OPTIONS]
```
##### Options
| Name, shorthand | Default| Description|
| -------------   |:------:|:----------|
| --id, -i        |        | Application id |
| --reset, -r     |        | Reset the active application |
| --json, -j      |        | Render result in JSON format |

##### Example
```sh
$ espercli app active -i 630dbfab-7d85-4f81-9f3b-ffb038b0df72

TITLE             DETAILS
id                630dbfab-7d85-4f81-9f3b-ffb038b0df72
application_name  Root Checker Basic
package_name      com.joeykrim.rootcheck
developer
category
content_rating    0.0
compatibility
version_count     1
```

### **Version**
Version command used to list, show and delete application versions.
```sh
$ espercli version [SUB-COMMANDS]
```
#### Sub commands
#### 1. list
List all application versions and can filter results by using different options listed below. Pagination used to limit the number of results, default is 20 results per page. Active application is used to list if `--app` or `-a` option is not given explicitly.
```sh
$ espercli version list [OPTIONS]
```
##### Options
| Name, shorthand | Default| Description|
| -------------   |:------:|:----------|
| --limit, -l     |20      | Number of results to return per page |
| --offset, -i    |0       | The initial index from which to return the results |
| --app, -a       |        | Application id (UUID) |
| --code, -c      |        | Filter by version code |
| --number, -n    |        | Filter by build number |
| --json, -j      |        | Render result in JSON format |

##### Example
```sh
$ espercli version list -a 630dbfab-7d85-4f81-9f3b-ffb038b0df72
Total Number of Versions: 1

ID                                    VERSION CODE      BUILD NUMBER    SIZE IN MB  RELEASE TRACK      INSTALLED COUNT
54436edb-9b43-4e2c-8107-2c6fa90e2a9e  6.4.5                      189       9.36421                                   1
```
For list of versions if active application is set,
```sh
$ espercli version list
Total Number of Versions: 1

ID                                    VERSION CODE      BUILD NUMBER    SIZE IN MB  RELEASE TRACK      INSTALLED COUNT
54436edb-9b43-4e2c-8107-2c6fa90e2a9e  6.4.5                      189       9.36421                                   1
```

#### 2. show
Show application version information, here version id (UUID) is required to show version information.
```sh
$ espercli version show [OPTIONS] [version-id]
```

##### Options
| Name, shorthand | Default| Description|
| -------------   |:------:|:----------|
| --app, -a       |        | Application id (UUID) |
| --json, -j      |        | Render result in JSON format |

##### Example
```sh
$ espercli version show -a 630dbfab-7d85-4f81-9f3b-ffb038b0df72 54436edb-9b43-4e2c-8107-2c6fa90e2a9e

TITLE            DETAILS
id               54436edb-9b43-4e2c-8107-2c6fa90e2a9e
version_code     6.4.5
build_number     189
size_in_mb       9.36421394348145
release_track
installed_count  1
```

#### 3. delete
Delete sub command used to delete particular application version. Here, version id (UUID) is required to delete version. Application will be also deleted if application contains one version and reset active application if it is set as active
```sh
$ espercli version delete [OPTIONS] [version-id]
```

##### Options
| Name, shorthand | Default| Description|
| -------------   |:------:|:----------|
| --app, -a       |        | Application id (UUID) |

##### Example
```sh
$ espercli version delete -a 630dbfab-7d85-4f81-9f3b-ffb038b0df72 54436edb-9b43-4e2c-8107-2c6fa90e2a9e
Version with id 54436edb-9b43-4e2c-8107-2c6fa90e2a9e deleted successfully
```

### **Device-command**
Device-command command used to fire different actions on device like lock, ping, reboot, deploy application and wipe.
```sh
$ espercli device-command [SUB-COMMANDS]
```
#### Sub commands
#### 1. install
Deploy an application version on device. Active device is used to install application if `--device` or `-d` option is not given explicitly.
```sh
$ espercli device-command install [OPTIONS]
```
##### Options
| Name, shorthand | Default| Description|
| -------------   |:------:|:----------|
| --device, -d    |        | Device name |
| --version, -v   |        | Application version id (UUID) |
| --json, -j      |        | Render result in JSON format |

##### Example
```sh
$ espercli device-command install -d SNA-SNL-3GQA -v 54436edb-9b43-4e2c-8107-2c6fa90e2a9e

TITLE    DETAILS
id       21180eef-678f-4447-87d8-e29af2bcb8e6
command  INSTALL
state    Command Initiated
```
For install command if active device is set,
```sh
$ espercli device-command install -v 54436edb-9b43-4e2c-8107-2c6fa90e2a9e

TITLE    DETAILS
id       21180eef-678f-4447-87d8-e29af2bcb8e6
command  INSTALL
state    Command Initiated
```

#### 2. uninstall
Uninstall an application version on device. Active device is used to uninstall application if `--device` or `-d` option is not given explicitly.
```sh
$ espercli device-command uninstall [OPTIONS]
```
##### Options
| Name, shorthand | Default| Description|
| -------------   |:------:|:----------|
| --device, -d    |        | Device name |
| --version, -v   |        | Application version id (UUID) |
| --json, -j      |        | Render result in JSON format |

##### Example
```sh
$ espercli device-command uninstall -d SNA-SNL-3GQA -v 54436edb-9b43-4e2c-8107-2c6fa90e2a9e

TITLE    DETAILS
id       21180eef-678f-4447-87d8-e29af2bcb8e6
command  UNINSTALL
state    Command Initiated
```

#### 3. ping
Ping a device, active device is used to ping if `--device` or `-d` option is not given explicitly.
```sh
$ espercli device-command ping [OPTIONS]
```
##### Options
| Name, shorthand | Default| Description|
| -------------   |:------:|:----------|
| --device, -d    |        | Device name |
| --json, -j      |        | Render result in JSON format |

##### Example
```sh
$ espercli device-command ping -d SNA-SNL-3GQA

TITLE    DETAILS
id       60f3f989-d59d-4c77-b4d9-aec385bd81fb
command  UPDATE_HEARTBEAT
state    Command Initiated
```

#### 4. lock
Lock command is used to lock screen of a device, active device is used to lock if `--device` or `-d` option is not given explicitly.
```sh
$ espercli device-command lock [OPTIONS]
```
##### Options
| Name, shorthand | Default| Description|
| -------------   |:------:|:----------|
| --device, -d    |        | Device name |
| --json, -j      |        | Render result in JSON format |

##### Example
```sh
$ espercli device-command lock -d SNA-SNL-3GQA

TITLE    DETAILS
id       6e00220d-9bc2-4176-839a-fb690f72f6e2
command  LOCK
state    Command Initiated
```

#### 5. reboot
Reboot command is used to reboot a device, active device is used to lock if `--device` or `-d` option is not given explicitly.
```sh
$ espercli device-command reboot [OPTIONS]
```
##### Options
| Name, shorthand | Default| Description|
| -------------   |:------:|:----------|
| --device, -d    |        | Device name |
| --json, -j      |        | Render result in JSON format |

##### Example
```sh
$ espercli device-command reboot -d SNA-SNL-3GQA

TITLE    DETAILS
id       6e00220d-9bc2-4176-839a-fb690f72f165
command  REBOOT
state    Command Initiated
```

#### 6. wipe
Wipe a device, active device is used to wipe if `--device` or `-d` option is not given explicitly.
```sh
$ espercli device-command wipe [OPTIONS]
```
##### Options
| Name, shorthand | Default| Description|
| -------------   |:------:|:----------|
| --device, -d    |        | Device name |
| --exstorage, -e |        | External storage needed to wipe or not |
| --frp, -f       |        | Factory reset production enabled or not |
| --json, -j      |        | Render result in JSON format |

##### Example
```sh
$ espercli device-command wipe -d SNA-SNL-3GQA -e -f

TITLE    DETAILS
id       8000220d-9bc2-4176-839a-fb690f72f165
command  WIPE
state    Command Initiated
```

#### 7. show
Show device-command information and command id (UUID) is required to show command information. This is used active device to show command if `--device` or `-d` option is not given explicitly.
```sh
$ espercli device-command show [OPTIONS] [command-id]
```
##### Options
| Name, shorthand | Default| Description|
| -------------   |:------:|:----------|
| --device, -d    |        | Device name |
| --json, -j      |        | Render result in JSON format |

##### Example
```sh
$ espercli device-command show 6e00220d-9bc2-4176-839a-fb690f72f6e2 -d SNA-SNL-3GQA

TITLE    DETAILS
id       60f3f989-d59d-4c77-b4d9-aec385bd81fb
command  UPDATE_HEARTBEAT
state    Command Success
```

### **Group-command**
Group-command command used to fire different actions on group like lock, ping, reboot and deploy application.
```sh
$ espercli device-command [SUB-COMMANDS]
```
#### Sub commands
#### 1. install
Deploy an application version on a group. Active group is used to install application if `--group` or `-g` option is not given explicitly.
```sh
$ espercli group-command install [OPTIONS]
```
##### Options
| Name, shorthand | Default| Description|
| -------------   |:------:|:----------|
| --group, -g     |        | Group name |
| --version, -v   |        | Application version id (UUID) |
| --json, -j      |        | Render result in JSON format |

##### Example
```sh
$ espercli group-command install -g 5G -v 54436edb-9b43-4e2c-8107-2c6fa90e2a9e

TITLE        DETAILS
id           6cda46b4-05da-4e76-b7ae-4af52ce288fa
command      INSTALL
state        Command Initiated
success
failed
in_progress
inactive
```
For list of versions if active group is set,
```sh
$ espercli group-command install -v 54436edb-9b43-4e2c-8107-2c6fa90e2a9e

TITLE        DETAILS
id           6cda46b4-05da-4e76-b7ae-4af52ce288fa
command      INSTALL
state        Command Initiated
success
failed
in_progress
inactive
```

#### 2. ping
Ping a group, active group is used to ping if `--group` or `-g` option is not given explicitly.
```sh
$ espercli group-command ping [OPTIONS]
```
##### Options
| Name, shorthand | Default| Description|
| -------------   |:------:|:----------|
| --group, -g     |        | Group name |
| --json, -j      |        | Render result in JSON format |

##### Example
```sh
$ espercli group-command ping -g 5G

TITLE        DETAILS
id           077b202f-d515-45bd-9764-8f9b42416959
command      UPDATE_HEARTBEAT
state        Command Initiated
success
failed
in_progress
inactive
```

#### 3. lock
Lock command is used to lock screen of a group of devices, active group is used to lock if `--group` or `-g` option is not given explicitly.
```sh
$ espercli group-command lock [OPTIONS]
```
##### Options
| Name, shorthand | Default| Description|
| -------------   |:------:|:----------|
| --group, -g     |        | Group name |
| --json, -j      |        | Render result in JSON format |

##### Example
```sh
$ espercli group-command lock -g 5G

TITLE        DETAILS
id           4752969d-b51f-410f-8b3b-956db59f8a61
command      LOCK
state        Command Initiated
success
failed
in_progress
inactive
```

#### 4. reboot
Reboot command is used to reboot group of devices, active group is used to lock if `--group` or `-g` option is not given explicitly.
```sh
$ espercli group-command reboot [OPTIONS]
```
##### Options
| Name, shorthand | Default| Description|
| -------------   |:------:|:----------|
| --group, -g     |        | Group name |
| --json, -j      |        | Render result in JSON format |

##### Example
```sh
$ espercli group-command reboot -g 5G

TITLE        DETAILS
id           b55d18ab-ff92-405b-8598-373594dd394e
command      REBOOT
state        Command Initiated
success
failed
in_progress
inactive
```

#### 5. show
Show group-command information and command id (UUID) is required to show command information. This is used active group to show command if `--group` or `-g` option is not given explicitly.
```sh
$ espercli group-command show [OPTIONS] [command-id]
```
##### Options
| Name, shorthand | Default| Description|
| -------------   |:------:|:----------|
| --group, -g     |        | Group name |
| --json, -j      |        | Render result in JSON format |

##### Example
```sh
$ espercli group-command show -g 5G b55d18ab-ff92-405b-8598-373594dd394e

TITLE        DETAILS
id           b55d18ab-ff92-405b-8598-373594dd394e
command      REBOOT
state        Command Success
success      SNA-SNL-73YE
             SNA-SNL-NYWL
failed
in_progress
inactive     
```
 
 ### **Installs**
Installs command used to list all installations on a device.
```sh
$ espercli installs [SUB-COMMANDS]
```
#### Sub commands
#### 1. list
List all application installations on a device and can filter results by using different options listed below. Pagination used to limit the number of results, default is 20 results per page. Active device is used to list if `--device` or `-d` option is not given explicitly.
```sh
$ espercli installs list [OPTIONS]
```
##### Options
| Name, shorthand | Default| Description|
| -------------   |:------:|:----------|
| --limit, -l     |20      | Number of results to return per page |
| --offset, -i    |0       | The initial index from which to return the results |
| --device, -d    |        | Device name |
| --appname, -an  |        | Application name |
| --package, -p   |        | Application package name |
| --state, -s     |        | Install state. Values are [Installation In-Progress, Uninstallation In-Progress, Install Success, Install Failed, Uninstall Success, Uninstall Failed] |
| --json, -j      |        | Render result in JSON format |

##### Example
 ```sh
 $ espercli installs list -d SNA-SNL-3GQA
 Total Number of Installs: 1

ID                                    APPLICATION         PACKAGE                 VERSION            STATE
fc9e0d4e-fc88-4729-a575-7d4645901f1d  Root Checker Basic  com.joeykrim.rootcheck  6.4.5              Install Success
 ```
 
 ### **status**
Status command used to list latest device event information.
```sh
$ espercli status [SUB-COMMANDS]
```
#### Sub commands
#### 1. latest
Show latest device event, active device is used to list if `--device` or `-d` option is not given explicitly.
```sh
$ espercli status latest [OPTIONS]
```
##### Options
| Name, shorthand | Default| Description|
| -------------   |:------:|:----------|
| --device, -d    |        | Device name |
| --json, -j      |        | Render result in JSON format |

##### Example
 ```sh
 $ espercli status latest -d SNA-SNL-3GQA

TITLE                  DETAILS
battery_level               98
battery_temperature         25
data_download           366914
data_upload             130956
memory_storage            8294
memory_ram                1177
link_speed                  65
signal_strength              2
 ```
 
We are always in active development and we try our best to keep our documentation up to date. However, if you end up ahead of time you can check our latest documentation on [Github](https://github.com/esper-io/esper-cli).

If you face any issue with CLI usage, we recommend you to reach out to [Developer Support](http://example.com)
