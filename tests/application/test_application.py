import os
from unittest import TestCase

from _pytest.monkeypatch import MonkeyPatch

from esper.main import EsperTest
from tests.utils import set_configure, teardown


class ApplicationTest(TestCase):

    def setUp(self) -> None:
        self.monkeypatch = MonkeyPatch()
        set_configure(self.monkeypatch)

    def tearDown(self) -> None:
        teardown()

    def test_list_application(self):
        argv = ['app', 'upload', 'tests/application/Tiny Notepad Simple Small_v1.0_apkpure.com.apk']
        with EsperTest(argv=argv) as app:
            app.run()

        argv = ['app', 'list', '--name', 'Tiny Notepad']
        with EsperTest(argv=argv) as app:
            app.run()
            data, output = app.last_rendered

            application_id = data[0]["ID"]
            assert len(data) == 1
            assert data[0]["NAME"] == "Tiny Notepad"

        argv = ['app', 'delete', application_id]
        with EsperTest(argv=argv) as app:
            app.run()

    def test_show_application(self):
        argv = ['app', 'upload', 'tests/application/Tiny Notepad Simple Small_v1.0_apkpure.com.apk']
        with EsperTest(argv=argv) as app:
            app.run()
            data, output = app.last_rendered
            application_id = data[0]["DETAILS"]

        argv = ['app', 'show', application_id]
        with EsperTest(argv=argv) as app:
            app.run()
            data, output = app.last_rendered

            assert data[0]["DETAILS"] == application_id
            assert data[1]["DETAILS"] == "Tiny Notepad"

        argv = ['app', 'delete', application_id]
        with EsperTest(argv=argv) as app:
            app.run()

    def test_show_application_with_active(self):
        argv = ['app', 'upload', 'tests/application/Tiny Notepad Simple Small_v1.0_apkpure.com.apk']
        with EsperTest(argv=argv) as app:
            app.run()
            data, output = app.last_rendered
            application_id = data[0]["DETAILS"]

        argv = ['app', 'show', application_id, '--active']
        with EsperTest(argv=argv) as app:
            app.run()

        argv = ['app', 'set-active']
        with EsperTest(argv=argv) as app:
            app.run()
            data, output = app.last_rendered

            assert data[0]["DETAILS"] == application_id
            assert data[1]["DETAILS"] == "Tiny Notepad"

        argv = ['app', 'delete', application_id]
        with EsperTest(argv=argv) as app:
            app.run()

    def test_upload_application(self):
        argv = ['app', 'upload', 'tests/application/Tiny Notepad Simple Small_v1.0_apkpure.com.apk']
        with EsperTest(argv=argv) as app:
            app.run()
            data, output = app.last_rendered
            application_id = data[0]["DETAILS"]

            assert data[1]["DETAILS"] == "Tiny Notepad"

        argv = ['app', 'delete', application_id]
        with EsperTest(argv=argv) as app:
            app.run()

    def test_upload_invalid_application_file(self):
        file = open("file.txt", "w+")
        file.close()

        argv = ['app', 'upload', 'file.txt']
        with EsperTest(argv=argv) as app:
            app.run()
            data, output = app.last_rendered

            assert data == "ERROR: app_file: The submitted file is empty. "

        os.remove('file.txt')

    def test_upload_application_file_exists(self):
        argv = ['app', 'upload', 'tests/application/Tiny Notepad Simple Small_v1.0_apkpure.com.apk']
        with EsperTest(argv=argv) as app:
            app.run()
            data, output = app.last_rendered
            application_id = data[0]["DETAILS"]

        argv = ['app', 'upload', 'tests/application/Tiny Notepad Simple Small_v1.0_apkpure.com.apk']
        with EsperTest(argv=argv) as app:
            app.run()
            data, output = app.last_rendered

            assert data == "ERROR: App version already exists "

        argv = ['app', 'delete', application_id]
        with EsperTest(argv=argv) as app:
            app.run()

    def test_delete_application(self):
        argv = ['app', 'upload', 'tests/application/Tiny Notepad Simple Small_v1.0_apkpure.com.apk']
        with EsperTest(argv=argv) as app:
            app.run()
            data, output = app.last_rendered
            application_id = data[0]["DETAILS"]

        argv = ['app', 'delete', application_id]
        with EsperTest(argv=argv) as app:
            app.run()
            data, output = app.last_rendered

            assert data == f"Application with id {application_id} deleted successfully"

    def test_set_active_application(self):
        argv = ['app', 'upload', 'tests/application/Tiny Notepad Simple Small_v1.0_apkpure.com.apk']
        with EsperTest(argv=argv) as app:
            app.run()
            data, output = app.last_rendered
            application_id = data[0]["DETAILS"]

        argv = ['app', 'set-active', '--id', application_id]
        with EsperTest(argv=argv) as app:
            app.run()
            data, output = app.last_rendered

            assert data[0]["DETAILS"] == application_id
            assert data[1]["DETAILS"] == "Tiny Notepad"

        argv = ['app', 'delete', application_id]
        with EsperTest(argv=argv) as app:
            app.run()

    def test_unset_active_application(self):
        argv = ['app', 'upload', 'tests/application/Tiny Notepad Simple Small_v1.0_apkpure.com.apk']
        with EsperTest(argv=argv) as app:
            app.run()
            data, output = app.last_rendered
            application_id = data[0]["DETAILS"]

        argv = ['app', 'set-active', '--id', application_id]
        with EsperTest(argv=argv) as app:
            app.run()

        argv = ['app', 'unset-active']
        with EsperTest(argv=argv) as app:
            app.run()
            data, output = app.last_rendered

            assert data == f"Unset the active application {application_id}"

        argv = ['app', 'delete', application_id]
        with EsperTest(argv=argv) as app:
            app.run()

    def test_show_active_application(self):
        argv = ['app', 'upload', 'tests/application/Tiny Notepad Simple Small_v1.0_apkpure.com.apk']
        with EsperTest(argv=argv) as app:
            app.run()
            data, output = app.last_rendered
            application_id = data[0]["DETAILS"]

        argv = ['app', 'set-active', '--id', application_id]
        with EsperTest(argv=argv) as app:
            app.run()

        argv = ['app', 'set-active']
        with EsperTest(argv=argv) as app:
            app.run()
            data, output = app.last_rendered

            assert data[0]["DETAILS"] == application_id
            assert data[1]["DETAILS"] == "Tiny Notepad"

        argv = ['app', 'delete', application_id]
        with EsperTest(argv=argv) as app:
            app.run()

    def test_download_application(self):
        argv = ['app', 'upload', 'tests/application/Tiny Notepad Simple Small_v1.0_apkpure.com.apk']
        with EsperTest(argv=argv) as app:
            app.run()
            data, output = app.last_rendered
            application_id = data[0]["DETAILS"]
            version_id = data[7]["DETAILS"]

        dest_file_name = 'testfile.apk'
        argv = ['app', 'download', version_id, '--app', application_id, '--dest', dest_file_name]
        with EsperTest(argv=argv) as app:
            app.run()

        assert os.path.exists(dest_file_name) == True

        if os.path.exists(dest_file_name):
            os.remove(dest_file_name)
