from unittest import TestCase

from _pytest.monkeypatch import MonkeyPatch

from esper.main import EsperTest
from tests.utils import set_configure, teardown


class GroupTest(TestCase):

    def setUp(self) -> None:
        self.monkeypatch = MonkeyPatch()
        set_configure(self.monkeypatch)

    def tearDown(self) -> None:
        teardown()

    def test_list_group(self):
        argv = ['group', 'list', '--name', 'All devices']
        with EsperTest(argv=argv) as app:
            app.run()
            data, output = app.last_rendered

            assert len(data) == 1
            assert data[0]["NAME"] == "All devices"

    def test_show_group(self):
        argv = ['group', 'show', 'All devices']
        with EsperTest(argv=argv) as app:
            app.run()
            data, output = app.last_rendered

            assert data[1]["DETAILS"] == "All devices"

    def test_show_not_existed_group(self):
        argv = ['group', 'show', 'wrong-group']
        with EsperTest(argv=argv) as app:
            app.run()
            data, output = app.last_rendered

            assert data == "Group does not exist with name wrong-group"

    def test_show_group_with_active(self):
        argv = ['group', 'show', 'All devices', '--active']
        with EsperTest(argv=argv) as app:
            app.run()

        argv = ['group', 'set-active']
        with EsperTest(argv=argv) as app:
            app.run()
            data, output = app.last_rendered

            assert data[1]["DETAILS"] == "All devices"

    def test_set_active_group(self):
        argv = ['group', 'set-active', '--name', 'All devices']
        with EsperTest(argv=argv) as app:
            app.run()
            data, output = app.last_rendered

            assert data[1]["DETAILS"] == "All devices"

    def test_unset_active_group(self):
        argv = ['group', 'set-active', '--name', 'All devices']
        with EsperTest(argv=argv) as app:
            app.run()

        argv = ['group', 'unset-active']
        with EsperTest(argv=argv) as app:
            app.run()
            data, output = app.last_rendered

            assert data == "Unset the active group All devices"

    def test_show_active_group(self):
        argv = ['group', 'set-active', '--name', 'All devices']
        with EsperTest(argv=argv) as app:
            app.run()

        argv = ['group', 'set-active']
        with EsperTest(argv=argv) as app:
            app.run()
            data, output = app.last_rendered

            assert data[1]["DETAILS"] == "All devices"

    def test_create_group_name_empty(self):
        argv = ['group', 'create']
        with EsperTest(argv=argv) as app:
            app.run()
            data, output = app.last_rendered

            assert data == "name cannot be empty."

    def test_create_group(self):
        argv = ['group', 'create', '--name', 'group1']
        with EsperTest(argv=argv) as app:
            app.run()
            data, output = app.last_rendered

            assert data[1]["DETAILS"] == "group1"

        argv = ['group', 'delete', 'group1']
        with EsperTest(argv=argv) as app:
            app.run()

    def test_create_group_name_aleady_exists(self):
        argv = ['group', 'create', '--name', 'All devices']
        with EsperTest(argv=argv) as app:
            app.run()
            data, output = app.last_rendered

            assert data == "ERROR: The fields name, enterprise must make a unique set. "

    def test_update_group(self):
        argv = ['group', 'create', '--name', 'group1']
        with EsperTest(argv=argv) as app:
            app.run()

        argv = ['group', 'update', 'group1', '--name', 'group2']
        with EsperTest(argv=argv) as app:
            app.run()
            data, output = app.last_rendered

            assert data[1]["DETAILS"] == "group2"

        argv = ['group', 'delete', 'group2']
        with EsperTest(argv=argv) as app:
            app.run()

    def test_delete_group(self):
        argv = ['group', 'create', '--name', 'group1']
        with EsperTest(argv=argv) as app:
            app.run()

        argv = ['group', 'delete', 'group1']
        with EsperTest(argv=argv) as app:
            app.run()
            data, output = app.last_rendered

            assert data == "Group with name group1 deleted successfully"

    def test_group_list_devices(self):
        argv = ['group', 'devices', '--group', 'All devices']
        with EsperTest(argv=argv) as app:
            app.run()
            data, output = app.last_rendered

            assert len(data) >= 0

    def test_group_add_devices(self):
        device = None
        argv = ['group', 'devices', '--group', 'All devices']
        with EsperTest(argv=argv) as app:
            app.run()
            data, output = app.last_rendered

            if len(data) > 0:
                device = data[0]["NAME"]

        if device:
            argv = ['group', 'create', '--name', 'group1']
            with EsperTest(argv=argv) as app:
                app.run()

            argv = ['group', 'add', '--group', 'group1', '--devices', device]
            with EsperTest(argv=argv) as app:
                app.run()
                data, output = app.last_rendered

                assert data[2]["DETAILS"] == 1

            argv = ['group', 'delete', 'group1']
            with EsperTest(argv=argv) as app:
                app.run()

    def test_group_remove_devices(self):
        device = None
        argv = ['group', 'devices', '--group', 'All devices']
        with EsperTest(argv=argv) as app:
            app.run()
            data, output = app.last_rendered

            if len(data) > 0:
                device = data[0]["NAME"]

        if device:
            argv = ['group', 'create', '--name', 'group1']
            with EsperTest(argv=argv) as app:
                app.run()

            argv = ['group', 'add', '--group', 'group1', '--devices', device]
            with EsperTest(argv=argv) as app:
                app.run()

            argv = ['group', 'remove', '--group', 'group1', '--devices', device]
            with EsperTest(argv=argv) as app:
                app.run()
                data, output = app.last_rendered

                assert data[2]["DETAILS"] == 0

            argv = ['group', 'delete', 'group1']
            with EsperTest(argv=argv) as app:
                app.run()
