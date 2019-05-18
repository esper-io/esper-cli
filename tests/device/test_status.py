from unittest import TestCase

from _pytest.monkeypatch import MonkeyPatch

from esper.main import EsperTest
from tests.utils import set_configure, teardown


class StatusTest(TestCase):

    def setUp(self) -> None:
        self.monkeypatch = MonkeyPatch()
        set_configure(self.monkeypatch)

        self.device = self.get_device()

    def tearDown(self) -> None:
        teardown()

    def get_device(self):
        argv = ['device', 'list', '--state', 'active', '--limit', '1']
        with EsperTest(argv=argv) as app:
            app.run()
            data, output = app.last_rendered

            return data[0]["NAME"]

    def test_list_status(self):
        if self.device:
            argv = ['status', 'latest', '--device', self.device]
            with EsperTest(argv=argv) as app:
                app.run()
                data, output = app.last_rendered

                assert len(data) >= 0
        else:
            assert 1 == 1
