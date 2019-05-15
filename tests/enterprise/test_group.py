from unittest import TestCase

from _pytest.monkeypatch import MonkeyPatch

from esper.main import EsperTest
from tests.utils import set_configure, teardown


class GroupTest(TestCase):

    def setUp(self) -> None:
        self.monkeypatch = MonkeyPatch()

    def tearDown(self) -> None:
        teardown()

    def test_list_group(self):
        set_configure(self.monkeypatch)
        argv = ['group', 'list', '--name', 'All devices']
        with EsperTest(argv=argv) as app:
            app.run()
            data, output = app.last_rendered

            assert len(data) == 1
            assert data[0]["NAME"] == "All devices"

    def test_show_group(self):
        set_configure(self.monkeypatch)
        argv = ['group', 'show', 'All devices']
        with EsperTest(argv=argv) as app:
            app.run()
            data, output = app.last_rendered

            assert data[1]["DETAILS"] == "All devices"

    def test_show_not_existed_group(self):
        set_configure(self.monkeypatch)
        argv = ['group', 'show', 'wrong group']
        with EsperTest(argv=argv) as app:
            app.run()
            data, output = app.last_rendered

            assert data == "ERROR: detail: Not found. "

    def test_show_group_with_active(self):
        set_configure(self.monkeypatch)
        argv = ['group', 'show', 'All devices', '--active']
        with EsperTest(argv=argv) as app:
            app.run()

        argv = ['group', 'active']
        with EsperTest(argv=argv) as app:
            app.run()
            data, output = app.last_rendered

            assert data[1]["DETAILS"] == "All devices"

    def test_set_active_group(self):
        set_configure(self.monkeypatch)
        argv = ['group', 'active', '--name', 'All devices']
        with EsperTest(argv=argv) as app:
            app.run()
            data, output = app.last_rendered

            assert data[1]["DETAILS"] == "All devices"

    def test_reset_active_group(self):
        set_configure(self.monkeypatch)
        argv = ['group', 'active', '--name', 'All devices']
        with EsperTest(argv=argv) as app:
            app.run()

        argv = ['group', 'active', '--reset']
        with EsperTest(argv=argv) as app:
            app.run()
            data, output = app.last_rendered

            assert data == "Reset the active group All devices"

    def test_show_active_group(self):
        set_configure(self.monkeypatch)
        argv = ['group', 'active', '--name', 'All devices']
        with EsperTest(argv=argv) as app:
            app.run()

        argv = ['group', 'active']
        with EsperTest(argv=argv) as app:
            app.run()
            data, output = app.last_rendered

            assert data[1]["DETAILS"] == "All devices"

    def test_create_group_name_empty(self):
        set_configure(self.monkeypatch)
        argv = ['group', 'create']
        with EsperTest(argv=argv) as app:
            app.run()
            data, output = app.last_rendered

            assert data == "name cannot be empty."

    def test_create_group(self):
        set_configure(self.monkeypatch)
        argv = ['group', 'create', '--name', 'group1']
        with EsperTest(argv=argv) as app:
            app.run()
            data, output = app.last_rendered

            assert data[1]["DETAILS"] == "group1"

        argv = ['group', 'delete', 'group1']
        with EsperTest(argv=argv) as app:
            app.run()

    def test_create_group_name_aleady_exists(self):
        set_configure(self.monkeypatch)
        argv = ['group', 'create', '--name', 'All devices']
        with EsperTest(argv=argv) as app:
            app.run()
            data, output = app.last_rendered

            assert data == "ERROR: The fields name, enterprise must make a unique set. "

    def test_update_group(self):
        set_configure(self.monkeypatch)
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
        set_configure(self.monkeypatch)
        argv = ['group', 'create', '--name', 'group1']
        with EsperTest(argv=argv) as app:
            app.run()

        argv = ['group', 'delete', 'group1']
        with EsperTest(argv=argv) as app:
            app.run()
            data, output = app.last_rendered

            assert data == "Group with name group1 deleted successfully"

    def test_group_devices(self):
        set_configure(self.monkeypatch)
        argv = ['group', 'devices', '--group', 'All devices']
        with EsperTest(argv=argv) as app:
            app.run()
            data, output = app.last_rendered

            assert len(data) > 1
