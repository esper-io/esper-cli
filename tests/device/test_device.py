from unittest import TestCase

from _pytest.monkeypatch import MonkeyPatch

from esper.main import EsperTest
from tests.utils import set_configure, teardown


class DeviceTest(TestCase):

    def setUp(self) -> None:
        self.monkeypatch = MonkeyPatch()
        set_configure(self.monkeypatch)
        self.device = self.get_device()

    def tearDown(self) -> None:
        teardown()

    def get_device(self):
        device = None
        argv = ['device', 'list']
        with EsperTest(argv=argv) as app:
            app.run()
            data, output = app.last_rendered

            if len(data) > 0:
                device = data[0]["NAME"]
        return device

    def test_list_device(self):
        if self.device:
            argv = ['device', 'list', '--name', self.device]
            with EsperTest(argv=argv) as app:
                app.run()
                data, output = app.last_rendered

                assert data[0]["NAME"] == self.device
        else:
            argv = ['device', 'list']
            with EsperTest(argv=argv) as app:
                app.run()
                data, output = app.last_rendered

                assert len(data) >= 0

    def test_show_device(self):
        if self.device:
            argv = ['device', 'show', self.device]
            with EsperTest(argv=argv) as app:
                app.run()
                data, output = app.last_rendered

                assert data[1]["DETAILS"] == self.device
        else:
            assert 1 == 1

    def test_show_not_existed_device(self):
        argv = ['device', 'show', 'wrong-device']
        with EsperTest(argv=argv) as app:
            app.run()
            data, output = app.last_rendered

            assert data == "Device does not exist with name wrong-device"

    def test_show_device_with_active(self):
        if self.device:
            argv = ['device', 'show', self.device, '--active']
            with EsperTest(argv=argv) as app:
                app.run()

            argv = ['device', 'set-active']
            with EsperTest(argv=argv) as app:
                app.run()
                data, output = app.last_rendered

                assert data[1]["DETAILS"] == self.device
        else:
            assert 1 == 1

    def test_set_active_device(self):
        if self.device:
            argv = ['device', 'set-active', '--name', self.device]
            with EsperTest(argv=argv) as app:
                app.run()
                data, output = app.last_rendered

                assert data[1]["DETAILS"] == self.device
        else:
            assert 1 == 1

    def test_unset_active_device(self):
        if self.device:
            argv = ['device', 'set-active', '--name', self.device]
            with EsperTest(argv=argv) as app:
                app.run()

            argv = ['device', 'unset-active']
            with EsperTest(argv=argv) as app:
                app.run()
                data, output = app.last_rendered

                assert data == f"Unset the active device {self.device}"
        else:
            assert 1 == 1

    def test_show_active_device(self):
        if self.device:
            argv = ['device', 'set-active', '--name', self.device]
            with EsperTest(argv=argv) as app:
                app.run()

            argv = ['device', 'set-active']
            with EsperTest(argv=argv) as app:
                app.run()
                data, output = app.last_rendered

                assert data[1]["DETAILS"] == self.device
        else:
            assert 1 == 1
