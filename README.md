# Esper CLI
[![Build Status](https://travis-ci.com/esper-io/esper-cli.svg?branch=master)](https://travis-ci.com/esper-io/esper-cli) [![Gitter](https://badges.gitter.im/esper-dev/esper-cli.svg)](https://gitter.im/esper-dev/esper-cli?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge)

This package provides a unified command line interface to the Esper API Services.

Current stable release versions are

    API version: 1.0.0
    SDK version: 0.0.12
    CLI version: 0.0.10

## Requirements

1. **Python:** You must use Python 3.6 or above.
2. **An Esper Dev Account:** You need a free Esper Dev Trial account to create an environment and generate an Esper `SERVER URL`to talk to APIs. You will choose the `ENVIRONMENT NAME` that will then be assigned as your custom URL and when you complete the sign up process your private environment will be created. See [Requesting an Esper Dev Trial account](https://docs.esper.io/home/gettingstarted.html#setup). 
3. **Generate an API key:** API key authentication is used for accessing APIs. You will have to generate this from the Esper Dev Console once you have set up your account. For example, the Esper Dev Console for your account can be accessed at `https://foo.espercloud.com` if you choose the `ENVIRONMENT NAME` of “foo”. See [Generating an API Key](https://docs.esper.io/home/module/genapikey.html)

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

Before using espercli, you need to tell it about your Esper credentials. For this you will need `ENVIRONMENT NAME` and `API KEY` as generated in [Requirements](#requirements) section.
The way to get started is to run the espercli configure command:
```sh
$ espercli configure
$ Environment name: foo
$ Esper API Key: LpDriKp7MWJiRGcwc8xzREeUj8OEFa
```
To list available commands, either run `espercli` with no parameters or execute `espercli --help`:
```sh
usage: espercli [-h] [-D] [-q] [-v]
                {group-command,group,enterprise,status,install,version,device-command,app,device,configure}
                ...

Esper CLI tool to manage resources on Esper.io API service

optional arguments:
  -h, --help            show this help message and exit
  -D, --debug           full application debug mode
  -q, --quiet           suppress all console output
  -v, --version         show program's version number and exit

sub-commands:
  {secureadb,group-command,group,enterprise,status,installs,version,device-command,app,device,configure}
    secureadb           Setup Secure ADB connection to Device
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
    telemetry           Get telemtery data for a device over a period

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
environment   foo
api_key  LpDriKp7MWJiRGcwc8xzREeUj8OEFa
```

### **Token**
Token command is used to show the information associated with the token.
```sh
$ espercli token [SUB-COMMANDS]
```
#### Sub commands
#### 1. show
Show token information.
```sh
$ espercli token show [OPTIONS]
```
##### Options
| Name, shorthand| Default| Description|
| ------------- |:-------------:|:-----|
| --json, -j     |  | Render result in JSON format |

##### Example
```sh
$ espercli token show

TITLE          DETAILS
Enterprise Id  f44373cb-1800-43c6-aab3-c81f8b1f435c
Token          U1XEFTNS1ujAMK2Q7Gl3hfPclCclhX
Expires On     2019-11-19 15:42:16.637203+00:00
Scope          ['read', 'write', 'update', 'introspection', 'sdk', 'register']
Created On     2019-08-20 08:02:16.640250+00:00
Updated On     2019-08-20 08:02:16.640275+00:00

$ espercli token show -j
{"Enterprise": "f44373cb-1800-43c6-aab3-c81f8b1f435c", "Developer App": "5b4ececb-b446-4f47-9e6f-0b47760763be", "Token": "U1XEFTNS1ujAMK2Q7Gl3hfPclCclhX", "Expires On": ["read", "write", "update", "introspection", "sdk", "register"], "Created On": "2019-08-20 08:02:16.640250+00:00", "Updated On": "2019-08-20 08:02:16.640275+00:00"}%
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
Enterprise Id    595a6107-b137-448d-b217-e20cc58ee84d
Name             Foo Enterprise
Registered Name  Foo Enterprise
Address          #123, Industrial Layout, Random Avenue
Location         Santa Clara, CA
Zip Code         12345
Email            contact@foo.io
Contact Person   Shiv Sundar
Contact Number   +145678901234

$ espercli enterprise show -j
{"Enterprise Id": "595a6107-b137-448d-b217-e20cc58ee84d", "Name": "Foo Enterprise", "Registered Name": "Foo Enterprise", "Address": "#123, Industrial Layout, Random Avenue", "Location": "Santa Clara, CA", "Zip Code": "12345", "Email": "contact@foo.io", "Contact Person": "Shiv Sundar", "Contact Number": "+145678901234"}%
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
Enterprise Id    595a6107-b137-448d-b217-e20cc58ee84d
Name             Foo Enterprise
Registered Name  Foo Enterprise
Address          #123, Industrial Layout, Random Avenue
Location         Santa Clara, CA
Zip Code         12345
Email            contact@foo.io
Contact Person   Muneer M
Contact Number   +145678901234
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
| --tags, -t      |        | Filter by a tag  |
| --search        |        | Search by device name, alias name or device id  |
| --serial, -se   |        | Filter by device serial number |
| --brand, -b     |        | Filter by device brand name |
| --gms, -gm      |        | Filter by GMS and non GMS flag, choices are [true, false] |
| --json, -j      |        | Render result in JSON format |

##### Example
```sh
$ espercli device list -gm false
Number of Devices: 10

ID                                    NAME          MODEL     CURRENT STATE  TAGS
62d42cff-6979-48ed-bedf-8b25052a74d0  SNA-SNL-FZH5  QUALCOMM  INACTIVE
9877c1f0-0435-4185-a41b-e896e33bd438  SNA-SNL-V84B  QUALCOMM  INACTIVE       kiosk, cust
1bab8bf7-4b12-426e-a35b-00a718ec3490  SNA-SNL-XA05  POSBANK   INACTIVE       
9cdb45ed-5bc7-433a-b08b-1c0cffffebec  SNA-SNL-N7XY  Esper     DISABLED
d89a88f3-de5c-4acc-9eae-0868bd2fad15  SNA-SNL-U1K1  EMDOOR    INACTIVE       EM
fc3af4e3-79f4-483f-986e-3af60bb58809  SNA-SNL-T1PX  Vertex    DISABLED       ModelV, Prod, Beta
e2a7d069-b536-4700-b07b-4db9d9d9236c  SNA-SNL-B424  Esper     INACTIVE
218b37c5-b1cf-4768-8340-b2bc5f701b54  SNA-SNL-BGD3  EMDOOR    INACTIVE
647ef365-0b68-4fbd-aa11-febe54d668b1  SNA-SNL-8QJG  Intel     INACTIVE
c7c0382e-b911-451a-9d62-54936622d3b3  SNA-SNL-R123  QUALCOMM  DISABLED
```

#### 2. show
Show the details of the device. Here, `device-name` is required to show device information. 
Use the `--active` or `-a` flag to mark this device as the active device. This will allow you to call further device commands without specifying the device.
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
alias_name     sample
suid           tFxCx1wRMnMIk7kO3GpkgX--VQEI_FQxC13D1Bh4yRA
api_level      28
template_name  NonGMS
is_gms         False
state          INACTIVE
tags           kiosk, cust
```

#### 3. set-active
The set-active sub command used to set a device as the active device and show the details of the current active device with no options.
```sh
$ espercli device set-active [OPTIONS]
```
##### Options
| Name, shorthand | Default| Description|
| -------------   |:------:|:----------|
| --name, -n      |        | Device name |
| --json, -j      |        | Render result in JSON format |

##### Example
```sh
$ espercli device set-active -n SNA-SNL-FZH5

TITLE          DETAILS
id             62d42cff-6979-48ed-bedf-8b25052a74d0
device_name    SNA-SNL-FZH5
alias_name    sample
suid           tFxCx1wRMnMIk7kO3GpkgX--VQEI_FQxC13D1Bh4yRA
api_level      28
template_name  NonGMS
is_gms         False
state          INACTIVE
tags           kiosk, cust
```

#### 4. unset-active
The unset-active sub command used to unset the current active device.
```sh
$ espercli device unset-active
```

##### Example
```sh
$ espercli device unset-active
Unset the active device SNA-SNL-FZH5
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
Show the details of the group. Here, `group-name` is required to show group information. 
Use the `--active` or `-a` flag to mark this group as the active group. This will allow you to call further group commands without specifying the group.
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

#### 3. set-active
The set-active sub command used to set a group as the active group and show the details of the current active group with no options.
```sh
$ espercli group set-active [OPTIONS]
```
##### Options
| Name, shorthand | Default| Description|
| -------------   |:------:|:----------|
| --name, -n      |        | Group name |
| --json, -j      |        | Render result in JSON format |

##### Example
```sh
$ espercli group set-active -n 5G

TITLE         DETAILS
id            2e5efca2-7776-442e-a5ef-c2758d4a45a3
name          5G
device_count  2
```

#### 4. unset-active
The unset-active sub command used to unset the current active group.
```sh
$ espercli group unset-active
```

##### Example
```sh
$ espercli group unset-active
Unset the active group 5G
```

#### 5. create
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

#### 6. update
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

#### 7. delete
Remove particular group.
```sh
$ espercli group delete [group-name]
```

##### Example
```sh
$ espercli group delete 5G
Group with name 5G deleted successfully
```

#### 8. add
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

#### 9. remove
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

#### 10. devices
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

ID                                    NAME          MODEL     CURRENT STATE  TAGS
3ebc3afd-249b-4f10-8561-fa1a9ddb1bb7  SNA-SNL-KX37  Shoonya   ACTIVE
c8efa083-f325-4e3b-8d20-71b7a2927ffb  SNA-SNL-3606  QUALCOMM  INACTIVE       kiosk
```

### **App**
App command used to list, show, upload and delete applications and set application as active for further commands.
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

ID                                    NAME                PACKAGE NAME
d7131f72-17e4-40e9-bb9e-28f1fad1f623  ATID Reader         com.atid.app.atx
0c067884-8d72-41b5-9ed7-3e6f1f62d99d  Call Blocker        com.sappalodapps.callblocker
630dbfab-7d85-4f81-9f3b-ffb038b0df72  Root Checker Basic  com.joeykrim.rootcheck
4baf7157-9fee-4dc5-ab3a-81dc983d7332  Castle Clash        com.igg.castleclash
09368a1b-a9cd-45bc-8824-7190bc0f6b7e  WiFiAnalyzer        com.vrem.wifianalyzer
```

#### 2. show
Show the details of an application. Here, `application-id` (UUID) is required to show application information.
Use the `--active` or `-a` flag to mark this application as the active application. This will allow you to call further app related commands without specifying the application.
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
$ espercli app upload ~/foo/com.joeykrim.rootcheck-v1.1.apk
Uploading......: 100%|███████████████████████████████████████████████████████████████████████████████████████████████████████████████████████████████▉| 196k/196k [00:11<00:00, 18.1kB/s, file=com.joeykrim.rootcheck-v1.1.apk]

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

#### 4. download
Download sub command used to download an application file to local system, here version id (UUID) is required to download the application version file.
```sh
$ espercli app download [OPTIONS] [version-id]
```
##### Options
| Name, shorthand | Default| Description|
| -------------   |:------:|:----------|
| --app, -a       |        | Application id (UUID) |
| --dest, -d      |        | Destination file path |
| --json, -j      |        | Render result in JSON format |

##### Example
```sh
$ espercli app download -a 630dbfab-7d85-4f81-9f3b-ffb038b0df72 -d ~/foo/com.joeykrim.rootcheck-v1.1.apk 54436edb-9b43-4e2c-8107-2c6fa90e2a9e
Downloading......:  100%|███████████████████████████████████████████████████████████████████████████████████████████████████████████████████████████████▉| 196k/196k [00:11<00:00, 18.1kB/s]
```

#### 5. delete
Delete sub command used to delete application. Here, application id (UUID) is required to delete application and unset active application if it is set as active.
```sh
$ espercli app delete [application-id]
```

##### Example
```sh
$ espercli app delete 630dbfab-7d85-4f81-9f3b-ffb038b0df72
Application with id 630dbfab-7d85-4f81-9f3b-ffb038b0df72 deleted successfully
```

#### 6. set-active
The set-active sub command used to set an application as active application and show active application information with no options.
```sh
$ espercli app set-active [OPTIONS]
```
##### Options
| Name, shorthand | Default| Description|
| -------------   |:------:|:----------|
| --id, -i        |        | Application id |
| --json, -j      |        | Render result in JSON format |

##### Example
```sh
$ espercli app set-active -i 630dbfab-7d85-4f81-9f3b-ffb038b0df72

TITLE             DETAILS
id                630dbfab-7d85-4f81-9f3b-ffb038b0df72
application_name  Root Checker Basic
package_name      com.joeykrim.rootcheck
developer
category
content_rating    0.0
compatibility
```
Below example listing versions of current active app,
```sh
$ espercli version list
Total Number of Versions: 1

ID                                    VERSION CODE      BUILD NUMBER    SIZE IN MB  RELEASE TRACK      INSTALLED COUNT
54436edb-9b43-4e2c-8107-2c6fa90e2a9e  6.4.5                      189       9.36421                                   1
```

#### 7. unset-active
The unset-active sub command used to unset the current active application.
```sh
$ espercli app unset-active
```

##### Example
```sh
$ espercli app unset-active
Unset the active application 630dbfab-7d85-4f81-9f3b-ffb038b0df72
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
Delete sub command used to delete particular application version. Here, version id (UUID) is required to delete version. Application will be also deleted if application contains one version and unset active application if it is set as active
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
| --version, -V   |        | Application version id (UUID) |
| --json, -j      |        | Render result in JSON format |

##### Example
```sh
$ espercli device-command install -d SNA-SNL-3GQA -V 54436edb-9b43-4e2c-8107-2c6fa90e2a9e

TITLE    DETAILS
id       21180eef-678f-4447-87d8-e29af2bcb8e6
command  INSTALL
state    Command Initiated
```
For install command if active device is set,
```sh
$ espercli device-command install -V 54436edb-9b43-4e2c-8107-2c6fa90e2a9e

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
| --version, -V   |        | Application version id (UUID) |
| --json, -j      |        | Render result in JSON format |

##### Example
```sh
$ espercli device-command uninstall -d SNA-SNL-3GQA -V 54436edb-9b43-4e2c-8107-2c6fa90e2a9e

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

#### 7. Clear app data
Clear app data, active device is used to clear app data if `--device` or `-d` option is not given explicitly.
```sh
$ espercli device-command clear-app-data [OPTIONS]
```
##### Options
| Name, shorthand | Default| Description|
| -------------   |:------:|:----------|
| --device, -d    |        | Device name |
| --package-name, -P |        | Package name of app to clear data from |
| --json, -j      |        | Render result in JSON format |

##### Example
```sh
$ espercli device-command clear-app-data -d SNA-SNL-3GQA -P com.google.android.gms.maps

TITLE    DETAILS
id       8000220d-9bc2-4176-839a-fb690f72f165
command  CLEAR_APP_DATA
state    Command Initiated
```

#### 8. show
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
$ espercli group-command [SUB-COMMANDS]
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
| --version, -V   |        | Application version id (UUID) |
| --json, -j      |        | Render result in JSON format |

##### Example
```sh
$ espercli group-command install -g 5G -V 54436edb-9b43-4e2c-8107-2c6fa90e2a9e

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
$ espercli group-command install -V 54436edb-9b43-4e2c-8107-2c6fa90e2a9e

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

### **secureadb**
Secureadb is a new feature that allows users to connect to their devices over Remote adb (using `adb-tools`), over the 
internet, securely.
```sh
$ espercli secureadb [SUB-COMMANDS]
```
#### Sub commands
#### 1. connect
Establish a secure connection to the device; active device is used to connect if `--device` or `-d` option is not given explicitly.

This will open a local endpoint on your machine, to which the `adb-tool` can connect.
This local service will relay the traffic, securing by TLS encryption, from adb-tools to the device.
This will also keep running until `adb disconnect` is run, or the application is quit by `Ctrl+C`. 
```sh
$ espercli secureadb connect [OPTIONS]
```
##### Options
| Name, shorthand | Default| Description|
| -------------   |:------:|:----------|
| --device, -d    |        | Device name |

##### Example
 ```sh
 $ espercli secureadb connect -d SNA-SNL-3GQA

Initiating Remote ADB Session. This may take a few seconds...

Secure ADB Client
Please connect ADB client to the following endpoint: 127.0.0.1 : 62945
If adb-tools is installed, please run the command below:
        adb connect 127.0.0.1:62945

Press Ctrl+C to quit!

 ```

### **telemetry**
This is used to view telemetry data for a device over a period
```sh
$ espercli telemetry [SUB-COMMANDS]
```
#### Sub commands
#### 1. get-data
It fetches the telemetry data for a specific device, for a specific metric aggregated by 
`hour` or `month` or `day` and is specified by `-p, --period flag`. The statistic function is specified by may be `avg`, `sum` or `count` 
and is specified by `-s, --statistic` flag. Metric format is `{category}-{metric_name}`. 
The timespan to fetch telemetry data can be specified using two ways: Using `-l, --last` option or
`-t, --to` and `-f, --from` combination. 

To use `-l, --last` option use a number to specify number of hours, days, months relative to now for which data is required.
To specify absolute date range use `-f, --from` and `-t, --to` combination.

Available metric names {category}-{metric_name}
    
    battery-level
    battery-temperature
    battery-voltage
    battery-capacity_count
    battery-current_avg
    battery-current
    battery-energy_count
    battery-low_indicator
    battery-present
    battery-level_absolute
    battery-scale
    battery-charge_time_remaining
    memory-available_ram_measured
    memory-available_internal_storage_measured
    memory-os_occupied_storage_measured
    wifi_network-signal_strength
    wifi_network-link_speed
    wifi_network-frequency	
    data_usage-total_data_download
    data_usage-total_data_upload`
          
```sh
$ espercli telemetry get-data [OPTIONS]
```
##### Options
| Name, shorthand | Default| Description|
| -------------   |:------:|:----------|
| --device, -d    |        | Device name |
| --metric, -m    |        | Metric in format {category}-{metric_name} |
| --from, -f      |   {2 days since now}     | Start date time of telemetry data |
| --period, -p    |   hour | Aggregation period |
| --statistic, -s |   avg  | Statistic function |
| --last, -l      |        | Relative time from now. Use -n for n hour\'s since or n days since |
| --to, -t        |  {now} | End date time of telemetry data |

##### Example
 ```sh
 $ espercli telemetry get-data -m battery-level -l 7 -p month -d DEV-ELOP-FXVW

Telemetry data for device DEV-ELOP-FXVD
Time                    Value
2019-12-01T00:00:00Z  69.2948

$ espercli telemetry get-data -d DEV-ELOP-FXVD -m battery-level -f 2019-12-15T12:36:36 -t 2019-12-17T12:36:36
Telemetry data for device DEV-ELOP-FXVD
Time                    Value
2019-12-16T04:00:00Z
2019-12-16T05:00:00Z  26.5333
2019-12-16T06:00:00Z  41.3333
2019-12-16T07:00:00Z  48.25
2019-12-16T08:00:00Z  52
2019-12-16T09:00:00Z  60
2019-12-16T11:00:00Z  65
2019-12-17T06:00:00Z  85.125
2019-12-17T07:00:00Z  82.4
```


## **Pipeline**
Pipelines is used to create workflows consisting of actions to such as APP-INSTALL/APP-UNINSTALL/etc.
```sh
$ espercli pipeline

usage: espercli pipeline [-h] {execute,stage,create,edit,remove,show} ...

Pipeline commands

optional arguments:
  -h, --help            show this help message and exit

sub-commands:
  {execute,stage,create,edit,remove,show}
    execute             execute controller
    stage               stage controller
    create              Create a pipeline
    edit                Edit a pipeline(s)
    remove              Remove a Pipeline
    show                List or Fetch a pipeline(s)

Usage: espercli pipeline
```
#### Sub commands
#### 1. create
Create a new Pipeline


```sh
$ espercli pipeline create [OPTIONS]
```
##### Options
| Name, shorthand | Default| Description      |
|:---------------:|:------:|:----------------:|
| --name, -n      | [opt] | Name of Pipeline |
| --desc          | [opt] | Description for Pipeline |

##### Example
 ```sh
 $ espercli pipeline create 

Name of the Pipeline: <Pipeline Name>
Description for this Pipeline [optional]: <BLah>
What type of trigger do you want for your Pipeline?
[1] NewAppVersionEvent
[2] Skip for now...

1
Enter the Application name: <Blah>
Enter the Package name: <com.blah>

 ```

#### 2. edit
Edit an existing Pipeline

```sh
$ espercli pipeline edit [OPTIONS]
```
##### Options
| Name, shorthand   | Default | Description |
|:-----------------:|:-------:|:-----------:|
| --name, -n        | [opt]   | Name of Pipeline |
| --desc            | [opt]   | Description for Pipeline |
| --pipeline-id, -p | [opt]   | Pipeline ID |

##### Example
 ```sh
 $ espercli pipeline edit 

Change the name of the Pipeline: <blah>
Change the description for this Pipeline [optional]: <blah>
Enter the Pipeline ID:
What type of trigger do you want for your Pipeline?
[1] NewAppVersionEvent
[2] Skip for now...


 ```

#### 3. remove
Remove an existing Pipeline

```sh
$ espercli pipeline remove [OPTIONS]
```
##### Options
| Name, shorthand   | Default | Description |
|:-----------------:|:-------:|:-----------:|
| --pipeline-id, -p | [opt]   | Pipeline ID |

##### Example
 ```sh
 $ espercli pipeline remove 

Enter the Pipeline ID: ...

 ```

#### 4. show
List one or all Pipelines

```sh
$ espercli pipeline show [OPTIONS]
```
##### Options
| Name, shorthand   | Default | Description |
|:-----------------:|:-------:|:-----------:|
| --pipeline-id, -p | [opt]   | Pipeline ID |

##### Example
 ```sh
 $ espercli pipeline show 

ID                                    NAME              DESCRIPTION    STAGES    VERSION  TRIGGER              TRIGGER-APP
db26ae3b-dd80-4b91-8a21-e6e7bf3099af  Jeryn CLI                           1          4     NewAppVersionEvent    Firefox
9c36afa8-b710-4c28-b72f-8555a69fd907  asdcafsd           asdasfd          0          1     NewAppVersionEvent    Candy Crush Saga
3c7fbc8a-c420-4e14-8036-a6ff4a7efb58  asd                asdas            1          1     NewAppVersionEvent    Candlei
 ```

## Pipeline Stages
These sub command are used to add various named Stages to the Pipeline. 
A Stage is a logical grouping for the various operations. Each stage has a `ordering` field
that tells in what order the stages have to processed by the pipeline. The pipeline orders the 
stages in ascending order of `ordering` value, and process accordingly.

Please note this command depends on an existing pipeline. A Pipeline needs to be created first, before 
the stage sub-commands can be run.


```sh
$ espercli pipeline stage

usage: espercli pipeline stage [-h] {operation,create,edit,remove,show} ...

Pipeline Stage commands

optional arguments:
  -h, --help            show this help message and exit

sub-commands:
  {operation,create,edit,remove,show}
    operation           operation controller
    create              Add a Stage
    edit                Edit a Stage
    remove              Remove a Stage
    show                List all Stages

Usage: espercli pipeline stage
```

#### Sub commands
#### 1. create
Add a new stage to an existing pipeline

```sh
$ espercli pipeline stage create [OPTIONS]
```
##### Options
| Name, shorthand   | Default | Description |
|:-----------------:|:-------:|:-----------:|
| --pipeline-id, -p | [opt]   | Pipeline ID |
| --name, -n        | [opt]   | Name of Stage |
| --desc            | [opt]   | Description for Stage |
| --ordering        | [opt]   | Ordering for stage |

##### Example
 ```sh
 $ espercli pipeline stage create 

Enter the Pipeline ID: <uuid of pipeline>
Name of the Stage: <name of stage>
Order of this Stage: 10
Description for this Stage [optional]:
Added Stage to Pipeline Successfully! Details:

TITLE        DETAILS
id           86cd9c84-59be-4c89-a609-d76f85e38d53
operations   []
client       f44373cb-1800-43c6-aab3-c81f8b1f435c
name         blah
description
created_on   2019-12-26T05:23:09.185749Z
updated_on   2019-12-26T05:23:09.185795Z
version      1
ordering     10
pipeline     904bf55d-f39f-4dc7-b085-014712c567fc
 ```
 
 #### 2. edit
Edit an existing stage

```sh
$ espercli pipeline stage edit [OPTIONS]
```
##### Options
| Name, shorthand   | Default | Description |
|:-----------------:|:-------:|:-----------:|
| --pipeline-id, -p | [opt]   | Pipeline ID |
| --stage-id, -s    | [opt]   | Stage ID |
| --name, -n        | [opt]   | Name of Stage |
| --desc            | [opt]   | Description for Stage |
| --ordering        | [opt]   | Ordering for stage |

##### Example
 ```sh
 $ espercli pipeline stage edit 

Enter the Pipeline ID: <uuid of pipeline>
Enter the Stage ID: <uuid>
Change the name of the Stage:
Change the description for this Stage [optional]:
Change the Ordering for this Stage [optional]:

TITLE        DETAILS
id           <uuid>
operations   []
client       <uuid>
name         blah
description
created_on   2019-12-26T05:23:09.185749Z
updated_on   2019-12-26T05:23:09.185795Z
version      1
ordering     10
pipeline     <uuid>

 ```

#### 3. remove
Remove an existing Stage

```sh
$ espercli pipeline stage remove [OPTIONS]
```
##### Options
| Name, shorthand   | Default | Description |
|:-----------------:|:-------:|:-----------:|
| --pipeline-id, -p | [opt]   | Pipeline ID |
| --stage-id, -s    | [opt]   | Stage ID |


##### Example
 ```sh
 $ espercli pipeline remove 

Enter the Pipeline ID: <uuid of pipeline>
Enter the Stage ID: <uuid>

 ```

#### 4. show
List one or all stages in a pipeline

```sh
$ espercli pipeline stage show [OPTIONS]
```
##### Options
| Name, shorthand   | Default | Description |
|:-----------------:|:-------:|:-----------:|
| --pipeline-id, -p | [opt]   | Pipeline ID |
| --stage-id, -s    | [opt]   | Stage ID    |

##### Example
 ```sh
 $ espercli pipeline stage show 

ID            NAME      DESCRIPTION   ORDERING    OPERATIONS  VERSION
<stage uuid>  2           2           2             0          1
<stage uuid>  3           3           3             0          1
<stage uuid>  blah                    10            0          1

 ```
 
## Pipeline Operations
These sub command are used to add various named Operations to the Stage of a Pipeline. 
An Operation defines an `Action` - such as `APP_INSTALL`, `APP_UNINSTALL`, ETC. Each operation has a `ordering` field, 
just like stages. The pipeline orders the operations in ascending order of `ordering` value, within a given stage, 
and processes them accordingly.

Note:
    If a `NewAppUploadEvent` trigger has been defined for the pipeline, then `APP_INSTALL`/`APP_UNINSTALL` operations, 
    will install/uninstall the app from the `NewAppUploadEvent`.

Please note this command depends on an existing Stage. A Stage needs to be created first, before the operations
sub-commands can be run.


```sh
$ espercli pipeline stage operation

usage: espercli pipeline stage operation [-h] {create,edit,remove,show} ...

Pipeline Stage Operation commands

optional arguments:
  -h, --help            show this help message and exit

sub-commands:
  {create,edit,remove,show}
    create              Add a Operation
    edit                Edit an Operation
    remove              Remove an Operation
    show                List all Operations

Usage: espercli pipeline stage operation
```

#### Sub commands
#### 1. create
Add a new Operation to an existing Stage

```sh
$ espercli pipeline stage operation create [OPTIONS]
```
##### Options
| Name, shorthand   | Default | Description |
|:-----------------:|:-------:|:-----------:|
| --pipeline-id, -p | [opt]   | Pipeline ID |
| --stage-id, -s    | [opt]   | Stage ID |
| --name, -n        | [opt]   | Name of Operation |
| --desc            | [opt]   | Description for Operation |
| --action, -a       | [opt]   | Action for Operation |
| --ordering        | [opt]   | Ordering for Operation |

##### Example
 ```sh
 $ espercli pipeline stage operation create 

Enter the Pipeline ID: <uuid of pipeline>
Enter the Stage ID: <uuid of stage>
Name of the Operation: <name of operation>
Action for this Operation:

1: App Install to a Group of Devices
2: App Uninstall to a Group of Devices
3: Reboot a Group of Devices

Enter the number for your selection: 1
Name of the Group (to which the command must be fired): <group-name>
Description for this Operation [optional]: 
Added Operation to Stage Successfully! Details:

TITLE        DETAILS
id           <operation uuid>
action       APP_INSTALL
action_args  {'url': '<group-command url>', 'body': {'command': 'INSTALL'}, 'method': 'POST', 'headers': {'Authorization': 'Bearer <oauth creds>'}}
client       f44373cb-1800-43c6-aab3-c81f8b1f435c
name         <name of operation>
description  
created_on   2019-12-26T06:19:50.329913Z
updated_on   2019-12-26T06:19:50.329944Z
version      1
ordering     1
pipeline     <pipeline uuid>
stage        <stage uuid>
 ```
 
 #### 2. edit
Edit an existing operation

```sh
$ espercli pipeline stage operation edit [OPTIONS]
```
##### Options
| Name, shorthand   | Default | Description |
|:-----------------:|:-------:|:-----------:|
| --pipeline-id, -p | [opt]   | Pipeline ID |
| --stage-id, -s    | [opt]   | Stage ID |
| --operation-id, -o| [opt]   | Operation ID |
| --name, -n        | [opt]   | Name of Operation |
| --desc            | [opt]   | Description for Operation |
| --action, -a       | [opt]   | Action for Operation |
| --ordering        | [opt]   | Ordering for Operation |

##### Example
 ```sh
 $ espercli pipeline stage operation edit 

Enter the Pipeline ID: <uuid of pipeline>
Enter the Stage ID: <uuid of stage>
Enter the Operation ID: <uuid>
Change the name of the Operation:
Change the description for this Operation [optional]:
Action for this Operation:

1: App Install to a Group of Devices
2: App Uninstall to a Group of Devices
3: Reboot a Group of Devices

Enter the number for your selection: 1
Edited Operation for this Stage Successfully! Details:

TITLE        DETAILS
id           <operation uuid>
action       APP_INSTALL
action_args  {'url': '<group-command url>', 'body': {'command': 'INSTALL'}, 'method': 'POST', 'headers': {'Authorization': 'Bearer <oauth creds>'}}
client       <uuid>
name         Jeryn
description  blah
created_on   2019-12-26T06:19:50.329913Z
updated_on   2019-12-26T07:00:54.485012Z
version      2
ordering     1
pipeline     <pipeline uuid>
stage        <stage uuid>
 ```

#### 3. remove
Remove an existing Stage

```sh
$ espercli pipeline stage remove [OPTIONS]
```
##### Options
| Name, shorthand   | Default | Description |
|:-----------------:|:-------:|:-----------:|
| --pipeline-id, -p | [opt]   | Pipeline ID |
| --stage-id, -s    | [opt]   | Stage ID |
| --operation-id, -o| [opt]   | Operation ID |


##### Example
 ```sh
 $ espercli pipeline stage operation remove 

Enter the Pipeline ID: <uuid of pipeline>
Enter the Stage ID: <uuid of stage>
Enter the Operation ID: <uuid>

 ```

#### 4. show
List one or all stages in a pipeline

```sh
$ espercli pipeline stage show [OPTIONS]
```
##### Options
| Name, shorthand   | Default | Description |
|:-----------------:|:-------:|:-----------:|
| --pipeline-id, -p | [opt]   | Pipeline ID |
| --stage-id, -s    | [opt]   | Stage ID    |
| --operation-id, -o| [opt]   | Operation ID |

##### Example
 ```sh
 $ espercli pipeline stage show 

Listing Operations for the Stage! Details:

ID                 NAME    DESCRIPTION      ORDERING  ACTION         VERSION
<operation uuid>    Jeryn   blah                    1  APP_INSTALL          2
 ``` 
 
## Pipeline Execution
This sub-command is used to manually start/stop/terminate a pipeline execution. 

Note:
    If trigger is specified for a pipeline, the execution will start automatically when
    trigger conditions are met.

```sh
$ espercli pipeline execute

usage: espercli pipeline execute [-h] {show,start,stop,terminate} ...

Pipeline Execute commands

optional arguments:
  -h, --help            show this help message and exit

sub-commands:
  {show,start,stop,terminate}
    show                List all Executions
    start               Execute pipeline
    stop                Stop a Pipeline Execution
    terminate           Terminate a Pipeline Execution

Usage: espercli pipeline execute
```

#### 1. show
List all Executions of a Pipeline

```sh
$ espercli pipeline execute show [OPTIONS]
```
##### Options
| Name, shorthand   | Default | Description |
|:-----------------:|:-------:|:-----------:|
| --pipeline-id, -p | [opt]   | Pipeline ID |

##### Example
 ```sh
 $ espercli pipeline execute show 

Enter the Pipeline ID: <uuid of pipeline>

Listing Operations for the Stage! Details:

ID                    NAME                    DESCRIPTION    STATE       STATUS      REASON
<pipeline uuid>       [DONT TOUCH] Jeryn CLI                 TERMINATED  TERMINATED  Dunno
<pipeline uuid>       [DONT TOUCH] Jeryn CLI                 COMPLETED   FAILURE     Out of 1 devices, Command failed on 1 devices, with 0 inactive devices
<pipeline uuid>       [DONT TOUCH] Jeryn CLI                 COMPLETED   FAILURE     Out of 1 devices, Command failed on 1 devices, with 0 inactive devices
 ```

#### 2. start
Start (or continue) a Manual execution of a Pipeline

```sh
$ espercli pipeline execute start [OPTIONS]
```
##### Options
| Name, shorthand   | Default | Description |
|:-----------------:|:-------:|:-----------:|
| --pipeline-id, -p | [opt]   | Pipeline ID |

##### Example
 ```sh
 $ espercli pipeline execute start 

Enter the Pipeline ID: <uuid of pipeline>

Pipeline execution started! Details:

TITLE        DETAILS
id           <execution uuid>
state        RUNNING
status       RUNNING
reason
client       <client uuid>
name         [DONT TOUCH] Jeryn CLI
description
version      4
started_at   2019-12-26T07:31:56.549188Z
updated_at   2019-12-26T07:31:56.654497Z
parent       <pipeline uuid>
 ```
 
#### 2. stop
Stop an execution of a Pipeline.

Note: A Stopped execution can be restarted with a `start` command

```sh
$ espercli pipeline execute stop [OPTIONS]
```
##### Options
| Name, shorthand   | Default | Description |
|:-----------------:|:-------:|:-----------:|
| --pipeline-id, -p | [opt]   | Pipeline ID |
| --execution-id, -e| [opt]   | Execution ID |
| --reason          | [opt]   | Reason to stop the execution |

##### Example
 ```sh
 $ espercli pipeline execute stop 

Enter the Pipeline ID: <uuid of pipeline>
Enter the Execution ID: <uuid of pipeline>
Why do you want to stop this Execution? : <give a reason>

Pipeline execution Stopped! Details:

TITLE        DETAILS
id           <execution uuid>
state        STOPPED
status       STOPPED
reason       <given reason>
client       <client uuid>
name         <pipeline name>
description
version      4
started_at   2019-12-26T07:31:56.549188Z
updated_at   2019-12-26T07:31:56.654497Z
parent       <pipeline uuid>
 ```

#### 3. terminate
Terminate an execution of a Pipeline.

Note: A terminated pipeline cant be restarted again.

```sh
$ espercli pipeline execute terminate [OPTIONS]
```
##### Options
| Name, shorthand   | Default | Description |
|:-----------------:|:-------:|:-----------:|
| --pipeline-id, -p | [opt]   | Pipeline ID |
| --execution-id, -e| [opt]   | Execution ID |
| --reason          | [opt]   | Reason to stop the execution |

##### Example
 ```sh
 $ espercli pipeline execute terminate 

Enter the Pipeline ID: <uuid of pipeline>
Enter the Execution ID: <uuid of pipeline>
Why do you want to terminate this Execution? : <give a reason>

Pipeline execution Stopped! Details:

TITLE        DETAILS
id           <execution uuid>
state        TERMINATED
status       TERMINATED
reason       <given reason>
client       <uuid>
name         <pipeline name>
description
version      4
started_at   2019-12-26T07:31:56.549188Z
updated_at   2019-12-26T07:31:56.654497Z
parent       <uuid>
 ```

#### 4. continue
Continue a Stopped execution of a Pipeline.

```sh
$ espercli pipeline execute continue [OPTIONS]
```
##### Options
| Name, shorthand   | Default | Description |
|:-----------------:|:-------:|:-----------:|
| --pipeline-id, -p | [opt]   | Pipeline ID |
| --execution-id, -e| [opt]   | Execution ID |

##### Example
 ```sh
 $ espercli pipeline execute continue 

Enter the Pipeline ID: <uuid of pipeline>
Enter the Execution ID: <uuid of pipeline>

Pipeline execution Started! Details:

TITLE        DETAILS
id           <execution uuid>
state        RUNNING
status       RUNNING
reason       <given reason>
client       <uuid>
name         <pipeline name>
description
version      4
started_at   2019-12-26T07:31:56.549188Z
updated_at   2019-12-26T07:31:56.654497Z
parent       <uuid>
 ```


We are always in active development and we try our best to keep our documentation up to date. However, if you end up ahead of time you can check our latest documentation on [Github](https://github.com/esper-io/esper-cli).

If you face any issue with CLI usage, we recommend you to reach out to [Esper Dev Support](https://docs.esper.io/home/support.html)
