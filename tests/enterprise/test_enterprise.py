from unittest import TestCase

from _pytest.monkeypatch import MonkeyPatch

from esper.main import EsperTest
from tests.utils import set_configure, teardown


class EnterpriseTest(TestCase):

    def setUp(self) -> None:
        self.monkeypatch = MonkeyPatch()
        set_configure(self.monkeypatch)

    def tearDown(self) -> None:
        teardown()

    def test_show_enterprise(self):
        argv = ['enterprise', 'show']
        with EsperTest(argv=argv) as app:
            app.run()
            data, output = app.last_rendered

            assert data[1]["DETAILS"] == "Shoonya Default Enterprise"
            assert data[3]["DETAILS"] == "Shoonya Default Enterprise"

    def test_update_enterprise(self):
        argv = ['enterprise', 'update', '--zipcode', '54321']
        with EsperTest(argv=argv) as app:
            app.run()
            data, output = app.last_rendered

            assert data[6]["DETAILS"] == "54321"

        # Revert the changes
        argv = ['enterprise', 'update', '--zipcode', '12345']
        with EsperTest(argv=argv) as app:
            app.run()
