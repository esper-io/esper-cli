import os
from os import path

from clint.textui import prompt

from esper.main import EsperTest


def get_esper_credentials():
    try:
        tenant = os.environ['ESPER_TENANT']
    except KeyError:
        print('ESPER_TENANT environment variable not set.')
        return

    try:
        key = os.environ['ESPER_KEY']
    except KeyError:
        print('ESPER_KEY environment variable not set.')
        return

    return {
        'tenant': tenant,
        'key': key
    }


def set_configure(monkeypatch):
    credentials = get_esper_credentials()

    def get_api_key(p):
        return credentials.get('key')

    def get_tenant(p):
        return credentials.get('tenant')

    monkeypatch.setattr(prompt, 'query', get_api_key)
    monkeypatch.setattr('builtins.input', get_tenant)

    argv = ['configure']
    with EsperTest(argv=argv) as app:
        app.run()


def teardown():
    if path.exists('creds.json'):
        os.remove('creds.json')
