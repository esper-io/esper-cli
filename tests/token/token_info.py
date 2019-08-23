from unittest import TestCase

from _pytest.monkeypatch import MonkeyPatch

from esper.main import EsperTest
from tests.utils import set_configure, teardown


class TokenTest(TestCase):

    def setUp(self) -> None:
        self.monkeypatch = MonkeyPatch()
        set_configure(self.monkeypatch)

    def tearDown(self) -> None:
        teardown()

    def test_show_token(self):
        argv = ['token', 'show']
        with EsperTest(argv=argv) as app:
            app.run()
            data, output = app.last_rendered
            assert data is not None
