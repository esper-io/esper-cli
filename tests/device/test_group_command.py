import time
from unittest import TestCase

from _pytest.monkeypatch import MonkeyPatch

from esper.controllers.enums import DeviceGroupCommandState, DeviceCommandState
from esper.main import EsperTest
from tests.utils import set_configure, teardown


class GroupCommandTest(TestCase):

    def setUp(self) -> None:
        self.monkeypatch = MonkeyPatch()
        set_configure(self.monkeypatch)

        self.group = self.create_group('group1')
        self.devices = self.get_devices()
        self.add_devices_to_group()

    def tearDown(self) -> None:
        argv = ['group', 'delete', self.group]
        with EsperTest(argv=argv) as app:
            app.run()

        teardown()

    def create_group(self, name):
        argv = ['group', 'create', '--name', name]
        with EsperTest(argv=argv) as app:
            app.run()
        return name

    def get_devices(self):
        devices = []
        argv = ['device', 'list', '--state', 'active', '--group', 'All devices', '--limit', '3']
        with EsperTest(argv=argv) as app:
            app.run()
            data, output = app.last_rendered
            if data:
                for i in range(0, len(data)):
                    if data[i]["CURRENT STATE"] == "ACTIVE":
                        devices.append(data[i]["NAME"])
        return devices

    def add_devices_to_group(self):
        if len(self.devices) > 0:
            argv = ['group', 'add', '--group', self.group, '--devices']
            argv.extend(self.devices)
            with EsperTest(argv=argv) as app:
                app.run()

    def is_command_completed(self, command):
        argv = ['group-command', 'show', '--group', self.group, command]
        with EsperTest(argv=argv) as app:
            app.run()
            data, output = app.last_rendered
            state = data[2]["DETAILS"]

        return state == DeviceGroupCommandState.SUCCESS.value or state == DeviceGroupCommandState.FAILURE.value

    def is_device_command_completed(self, device, command):
        argv = ['device-command', 'show', '--device', device, command]
        with EsperTest(argv=argv) as app:
            app.run()
            data, output = app.last_rendered
            state = data[2]["DETAILS"]

        return state == DeviceCommandState.SUCCESS.value or state == DeviceCommandState.FAILURE.value

    def get_installation_success_devices(self, command):
        devices = []
        argv = ['group-command', 'show', '--group', self.group, command]
        with EsperTest(argv=argv) as app:
            app.run()
            data, output = app.last_rendered
            state = data[3]["DETAILS"]

            if state:
                devices.extend(state.split('\n'))
        return devices

    def test_device_commands(self):
        if len(self.devices) == 0:
            return

        argv = ['app', 'upload', 'tests/application/Tiny Notepad Simple Small_v1.0_apkpure.com.apk']
        with EsperTest(argv=argv) as app:
            app.run()
            data, output = app.last_rendered
            application_id = data[0]["DETAILS"]
            version_id = data[7]["DETAILS"]

        argv = ['group-command', 'ping', '--group', self.group]
        with EsperTest(argv=argv) as app:
            app.run()
            data, output = app.last_rendered
            command_id = data[0]['DETAILS']

            assert data[1]["DETAILS"] == 'UPDATE_HEARTBEAT'
            assert data[2]["DETAILS"] == 'Command Initiated'

        while not self.is_command_completed(command_id):
            time.sleep(3)

        time.sleep(2)
        argv = ['group-command', 'lock', '--group', self.group]
        with EsperTest(argv=argv) as app:
            app.run()
            data, output = app.last_rendered
            command_id = data[0]['DETAILS']

            assert data[1]["DETAILS"] == 'LOCK'
            assert data[2]["DETAILS"] == 'Command Initiated'

        while not self.is_command_completed(command_id):
            time.sleep(3)

        time.sleep(2)
        argv = ['group-command', 'install', '--group', self.group, '--version', version_id]
        with EsperTest(argv=argv) as app:
            app.run()
            data, output = app.last_rendered
            command_id = data[0]['DETAILS']

            assert data[1]["DETAILS"] == 'INSTALL'
            assert data[2]["DETAILS"] == 'Command Initiated'

        while not self.is_command_completed(command_id):
            time.sleep(10)

        time.sleep(2)

        installed_devices = self.get_installation_success_devices(command_id)
        for device in installed_devices:
            argv = ['device-command', 'uninstall', '--device', device, '--version', version_id]
            with EsperTest(argv=argv) as app:
                app.run()
                data, output = app.last_rendered
                command_id = data[0]['DETAILS']

                while not self.is_device_command_completed(device, command_id):
                    time.sleep(3)

        time.sleep(2)
        argv = ['group-command', 'reboot', '--group', self.group]
        with EsperTest(argv=argv) as app:
            app.run()
            data, output = app.last_rendered
            command_id = data[0]['DETAILS']

            assert data[1]["DETAILS"] == 'REBOOT'
            assert data[2]["DETAILS"] == 'Command Initiated'

        while not self.is_command_completed(command_id):
            time.sleep(10)

        time.sleep(120)
        argv = ['app', 'delete', application_id]
        with EsperTest(argv=argv) as app:
            app.run()
