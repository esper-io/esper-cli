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


            assert data[2]["DETAILS"] == "Shoonya Default Enterprise"

    def test_update_enterprise(self):
        argv = ['enterprise', 'show', '--json']
        zipcode = None
        with EsperTest(argv=argv) as app:
            app.run()
            data, output = app.last_rendered
            zipcode =data['Zip Code']



        argv = ['enterprise', 'update', '--json','--zipcode', '54321']

        with EsperTest(argv=argv) as app:
            app.run()
            data, output = app.last_rendered

            assert data['Zip Code'] == '54321'

        # Revert the changes
        argv = ['enterprise', 'update', '--zipcode', zipcode]
        with EsperTest(argv=argv) as app:
            app.run()
