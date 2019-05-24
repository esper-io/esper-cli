from unittest import TestCase

from _pytest.monkeypatch import MonkeyPatch

from esper.main import EsperTest
from tests.utils import set_configure, teardown


class VersionTest(TestCase):

    def setUp(self) -> None:
        self.monkeypatch = MonkeyPatch()
        set_configure(self.monkeypatch)

        argv = ['app', 'upload', 'tests/application/Tiny Notepad Simple Small_v1.0_apkpure.com.apk']
        with EsperTest(argv=argv) as app:
            app.run()
            data, output = app.last_rendered
            self.application_id = data[0]["DETAILS"]

    def tearDown(self) -> None:
        argv = ['app', 'delete', self.application_id]
        with EsperTest(argv=argv) as app:
            app.run()

        teardown()

    def test_list_version(self):
        argv = ['version', 'list', '--app', self.application_id]
        with EsperTest(argv=argv) as app:
            app.run()
            data, output = app.last_rendered

            assert len(data) == 1

    def test_list_version_with_active_application(self):
        argv = ['app', 'set-active', '--id', self.application_id]
        with EsperTest(argv=argv) as app:
            app.run()

        argv = ['version', 'list']
        with EsperTest(argv=argv) as app:
            app.run()
            data, output = app.last_rendered

            assert len(data) == 1

    def test_show_version(self):
        argv = ['version', 'list', '--app', self.application_id,]
        with EsperTest(argv=argv) as app:
            app.run()
            data, output = app.last_rendered
            version_id = data[0]["ID"]

        argv = ['version', 'show', '--app', self.application_id, version_id]
        with EsperTest(argv=argv) as app:
            app.run()
            data, output = app.last_rendered

            assert data[0]["DETAILS"] == version_id

    def test_delete_version(self):
        argv = ['version', 'list', '--app', self.application_id,]
        with EsperTest(argv=argv) as app:
            app.run()
            data, output = app.last_rendered
            version_id = data[0]["ID"]

        argv = ['version', 'delete', '--app', self.application_id, version_id]
        with EsperTest(argv=argv) as app:
            app.run()
            data, output = app.last_rendered

            assert data == f"Version with id {version_id} deleted successfully"
