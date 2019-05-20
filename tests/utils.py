import os
from os import path

from clint.textui import prompt

from esper.main import EsperTest


def get_esper_credentials():
    try:
        environment = os.environ['ESPER_ENVIRONMENT']
    except KeyError:
        print('ESPER_ENVIRONMENT environment variable not set.')
        return

    try:
        key = os.environ['ESPER_KEY']
    except KeyError:
        print('ESPER_KEY environment variable not set.')
        return

    return {
        'environment': environment,
        'key': key
    }


def set_configure(monkeypatch):
    credentials = get_esper_credentials()

    def get_api_key(p):
        return credentials.get('key')

    def get_environment(p):
        return credentials.get('environment')

    monkeypatch.setattr(prompt, 'query', get_api_key)
    monkeypatch.setattr('builtins.input', get_environment)

    argv = ['configure']
    with EsperTest(argv=argv) as app:
        app.run()


def teardown():
    if path.exists('creds.json'):
        os.remove('creds.json')
