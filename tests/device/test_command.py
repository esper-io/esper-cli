import time
from unittest import TestCase

from _pytest.monkeypatch import MonkeyPatch

from esper.controllers.enums import DeviceCommandState
from esper.main import EsperTest
from tests.utils import set_configure, teardown


class CommandTest(TestCase):

    def setUp(self) -> None:
        self.monkeypatch = MonkeyPatch()
        set_configure(self.monkeypatch)

        self.device = self.get_device()

    def tearDown(self) -> None:
        teardown()

    def get_device(self):
        device = None
        argv = ['device', 'list', '--state', 'active', '--limit', '3']
        with EsperTest(argv=argv) as app:
            app.run()
            data, output = app.last_rendered

            if data:
                for i in range(0, len(data)):
                    if data[i]["CURRENT STATE"] == "ACTIVE":
                        device = data[i]["NAME"]
                        break
        return device

    def is_command_completed(self, command):
        argv = ['device-command', 'show', '--device', self.device, command]
        with EsperTest(argv=argv) as app:
            app.run()
            data, output = app.last_rendered
            state = data[2]["DETAILS"]

        return state == DeviceCommandState.SUCCESS.value or state == DeviceCommandState.FAILURE.value

    def test_device_commands(self):
        if self.device:
            argv = ['app', 'upload', 'tests/application/Tiny Notepad Simple Small_v1.0_apkpure.com.apk']
            with EsperTest(argv=argv) as app:
                app.run()
                data, output = app.last_rendered
                application_id = data[0]["DETAILS"]
                version_id = data[7]["DETAILS"]
                application_name = data[1]["DETAILS"]

            argv = ['device-command', 'ping', '--device', self.device]
            with EsperTest(argv=argv) as app:
                app.run()
                data, output = app.last_rendered
                command_id = data[0]['DETAILS']

                assert data[1]["DETAILS"] == 'UPDATE_HEARTBEAT'
                assert data[2]["DETAILS"] == 'Command Initiated'

            while not self.is_command_completed(command_id):
                time.sleep(3)

            time.sleep(2)
            argv = ['device-command', 'lock', '--device', self.device]
            with EsperTest(argv=argv) as app:
                app.run()
                data, output = app.last_rendered
                command_id = data[0]['DETAILS']

                assert data[1]["DETAILS"] == 'LOCK'
                assert data[2]["DETAILS"] == 'Command Initiated'

            while not self.is_command_completed(command_id):
                time.sleep(3)

            time.sleep(2)
            argv = ['device-command', 'install', '--device', self.device, '--version', version_id]
            with EsperTest(argv=argv) as app:
                app.run()
                data, output = app.last_rendered
                command_id = data[0]['DETAILS']

                assert data[1]["DETAILS"] == 'INSTALL'
                assert data[2]["DETAILS"] == 'Command Initiated'

            while not self.is_command_completed(command_id):
                time.sleep(10)

            argv = ['installs', 'list', '--device', self.device]
            with EsperTest(argv=argv) as app:
                app.run()
                data, output = app.last_rendered

                assert data[0]["APPLICATION"] == application_name
                assert data[0]["STATE"] == "Install Success"

            time.sleep(2)
            argv = ['device-command', 'uninstall', '--device', self.device, '--version', version_id]
            with EsperTest(argv=argv) as app:
                app.run()
                data, output = app.last_rendered
                command_id = data[0]['DETAILS']

                assert data[1]["DETAILS"] == 'UNINSTALL'
                assert data[2]["DETAILS"] == 'Command Initiated'

            while not self.is_command_completed(command_id):
                time.sleep(10)

            time.sleep(2)
            argv = ['device-command', 'reboot', '--device', self.device]
            with EsperTest(argv=argv) as app:
                app.run()
                data, output = app.last_rendered

                assert data[1]["DETAILS"] == 'REBOOT'
                assert data[2]["DETAILS"] == 'Command Initiated'

            while not self.is_command_completed(command_id):
                time.sleep(10)

            time.sleep(120)
            argv = ['app', 'delete', application_id]
            with EsperTest(argv=argv) as app:
                app.run()
        else:
            assert 1 == 1
