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

    INACTIVE = 60  # This state is set by cloud, when device is unreachable


class OutputFormat(BaseEnum):
    TABULATED = 'tabulated'
    JSON = 'json'


class DeviceCommandEnum(BaseEnum):
    INSTALL = 210
