from unittest import TestCase

from _pytest.monkeypatch import MonkeyPatch
from clint.textui import prompt

from esper.main import EsperTest
from tests.utils import get_esper_credentials, teardown, set_configure


class EnterpriseTest(TestCase):
    def setUp(self) -> None:
        self.monkeypatch = MonkeyPatch()

    def tearDown(self) -> None:
        teardown()

    def test_configure_valid_credential(self):
        credentials = get_esper_credentials()

        def get_api_key(p):
            return credentials.get('key')

        def get_tenant(p):
            return credentials.get('tenant')

        self.monkeypatch.setattr(prompt, 'query', get_api_key)
        self.monkeypatch.setattr('builtins.input', get_tenant)

        argv = ['configure']
        with EsperTest(argv=argv) as app:
            app.run()

            data, output = app.last_rendered

            assert data[0]["DETAILS"] == credentials.get('key')
            assert data[1]["DETAILS"] == credentials.get('tenant')

    def test_configure_invalid_credential(self):
        credentials = get_esper_credentials()

        def get_api_key(p):
            return 'wrongapikey'

        def get_tenant(p):
            return credentials.get('tenant')

        self.monkeypatch.setattr(prompt, 'query', get_api_key)
        self.monkeypatch.setattr('builtins.input', get_tenant)

        argv = ['configure']
        with EsperTest(argv=argv) as app:
            app.run()

            data, output = app.last_rendered
            assert data == "You are not authorized, invalid API Key."

    def test_set_configure(self):
        credentials = get_esper_credentials()

        def get_api_key(p):
            return credentials.get('key')

        def get_tenant(p):
            return credentials.get('tenant')

        self.monkeypatch.setattr(prompt, 'query', get_api_key)
        self.monkeypatch.setattr('builtins.input', get_tenant)

        argv = ['configure', '--set']
        with EsperTest(argv=argv) as app:
            app.run()

            data, output = app.last_rendered

            assert data[0]["DETAILS"] == credentials.get('key')
            assert data[1]["DETAILS"] == credentials.get('tenant')

    def test_show_configure(self):
        set_configure(self.monkeypatch)

        argv = ['configure']
        with EsperTest(argv=argv) as app:
            app.run()

            data, output = app.last_rendered

            credentials = get_esper_credentials()
            assert data[0]["DETAILS"] == credentials.get('key')
            assert data[1]["DETAILS"] == credentials.get('tenant')
