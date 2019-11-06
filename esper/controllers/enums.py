from enum import Enum


class BaseEnum(Enum):
    @classmethod
    def choices(cls):
        return [(member.value, name) for name, member in cls.__members__.items()]

    @classmethod
    def choice_list(cls):
        return list(cls.__members__.keys())


class DeviceState(BaseEnum):
    DEVICE_STATE_UNSPECIFIED = 0
    ACTIVE = 1
    DISABLED = 20
    PROVISIONING_BEGIN = 30
    GOOGLE_PLAY_CONFIGURATION = 40
    POLICY_APPLICATION_IN_PROGRESS = 50

    INACTIVE = 60


class OutputFormat(BaseEnum):
    TABULATED = 'tabulated'
    JSON = 'json'


class DeviceCommandEnum(BaseEnum):
    LOCK = 10
    REBOOT = 20
    UPDATE_HEARTBEAT = 40
    WIPE = 90
    INSTALL = 210
    UNINSTALL = 220
    CLEAR_APP_DATA = 260


class DeviceCommandState(BaseEnum):
    INITIATE = "Command Initiated"
    ACKNOWLEDGE = "Command Acknowledged"
    IN_PROGRESS = "Command In Progress"
    TIMEOUT = "Command TimeOut"
    SUCCESS = "Command Success"
    FAILURE = "Command Failure"
    SCHEDULED = "Command Scheduled"


class DeviceGroupCommandState(Enum):
    INITIATE = "Command Initiated"
    SUCCESS = "Command Success"
    FAILURE = "Command Failure"
    SCHEDULED = "Command Scheduled"

    @classmethod
    def choices(cls):
        return [(member.value, name) for name, member in cls.__members__.items()]
