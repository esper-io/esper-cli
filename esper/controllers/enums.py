from enum import Enum


class BaseEnum(Enum):
    @classmethod
    def choices(cls):
        return [(member.value, name) for name, member in cls.__members__.items()]

    @classmethod
    def choice_list(cls):
        return list(cls.__members__.keys())

    @classmethod
    def choice_list_lower(cls):
        return [val.lower() for val in cls.__members__.keys()]

class DeviceState(BaseEnum):
    DEVICE_STATE_UNSPECIFIED = 0
    ACTIVE = 1
    DISABLED = 20
    PROVISIONING_BEGIN = 30
    GOOGLE_PLAY_CONFIGURATION = 40
    POLICY_APPLICATION_IN_PROGRESS = 50

    INACTIVE = 60
    WIPE_IN_PROGRESS = 70

    ONBOARDING_IN_PROGRESS = 80
    ONBOARDING_FAILED = 90
    ONBOARDED = 100

    AFW_ACCOUNT_ADDED = 110
    APPS_INSTALLED = 120
    BRANDING_PROCESSED = 130
    PERMISSION_POLICY_PROCESSED = 140
    DEVICE_POLICY_PROCESSED = 150
    DEVICE_SETTINGS_PROCESSED = 160
    SECURITY_POLICY_PROCESSED = 170
    PHONE_POLICY_PROCESSED = 180
    CUSTOM_SETTINGS_PROCESSED = 190

    REGISTERED=200


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

class CommandEnum(BaseEnum):
    REBOOT = 20
    SET_NEW_POLICY = 30
    UPDATE_HEARTBEAT = 40
    WIPE = 90

    INSTALL = 210
    UNINSTALL = 220
    UPDATE_LATEST_DPC = 240

    SET_KIOSK_APP = 280

    SET_DEVICE_LOCKDOWN_STATE = 310
    SET_APP_STATE = 330
    ADD_WIFI_AP = 340
    REMOVE_WIFI_AP = 350
    UPDATE_DEVICE_CONFIG = 360

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

class CommandState(BaseEnum):
    QUEUED = "Command Queued"
    INITIATE = "Command Initiated"
    ACKNOWLEDGE = "Command Acknowledged"
    IN_PROGRESS = "Command In Progress"
    TIMEOUT = "Command TimeOut"
    SUCCESS = "Command Success"
    FAILURE = "Command Failure"
    SCHEDULED = "Command Scheduled"
    CANCELLED = "Command Cancelled"


class CommandScheduleEnum(BaseEnum):
    IMMEDIATE = "IMMEDIATE"
    WINDOW = "WINDOW"
    RECURRING = "RECURRING"


class CommandScheduleTimeTypeEnum(BaseEnum):
    CONSOLE = "console"
    DEVICE = "device"


class WeekDays(BaseEnum):
    SUNDAY = "Sunday"
    MONDAY = "Monday"
    TUESDAY = "Tuesday"
    WEDNESDAY = "Wednesday"
    THURSDAY = "Thursday"
    FRIDAY = "Friday"
    SATURDAY = "Saturday"
    ALL_DAYS = "All days"

class CommandRequestTypeEnum(BaseEnum):
    DEVICE = "DEVICE"
    GROUP = "GROUP"
    DYNAMIC = "DYNAMIC"


class CommandDeviceTypeEnum(BaseEnum):
    ACTIVE = "active"
    INACTIVE = "inactive"
    ALL = "all"
