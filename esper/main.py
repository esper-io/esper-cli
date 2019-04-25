from cement import App, TestApp, init_defaults
from cement.core.exc import CaughtSignal

from esper.controllers.base import Base
from esper.controllers.configure import Configure
from esper.controllers.devices import Devices
from esper.controllers.application import Application
from esper.controllers.command import Command
from esper.core.exc import EsperError
from esper.core.output_handler import EsperOutputHandler
from esper.ext.utils import extend_tinydb

# configuration defaults
CONFIG = init_defaults('esper')
CONFIG['esper']['debug'] = False
CONFIG['esper']['creds_file'] = '~/.esper/db/creds.json'

# meta defaults
META = init_defaults('log.colorlog')
META['log.colorlog']['file'] = '~/.esper/logs/esper.log'
META['log.colorlog']['level'] = 'debug'
META['log.colorlog']['to_console'] = True
META['log.colorlog']['rotate'] = False
META['log.colorlog']['max_bytes'] = 512000
META['log.colorlog']['max_files'] = 4
META['log.colorlog']['colorize_file_log'] = False
META['log.colorlog']['colorize_console_log'] = True
META['log.colorlog']['colors'] = {
    'DEBUG': 'cyan',
    'INFO': 'white',
    'WARNING': 'yellow',
    'ERROR': 'red',
    'CRITICAL': 'red,bg_white',
}


class Esper(App):
    """Esper CLI Tool primary application."""

    class Meta:
        label = 'esper'

        # call sys.exit() on close
        exit_on_close = True

        # configuration settings
        config_section = 'esper'
        config_defaults = CONFIG
        config_file_suffix = '.yml'
        config_files = [
            '~/.esper/config/esper.yml',
            '/etc/esper/esper.yml',
            '~/.config/esper/esper.yml',
            '~/.config/esper.yml',
            '~/.esper.yml'
        ]
        config_handler = 'yaml'

        meta_defaults = META
        # set the log handler
        log_handler = 'colorlog'

        # load additional framework extensions
        extensions = [
            'yaml',
            'json',
            'colorlog',
            'jinja2',
            'tabulate'
        ]

        # # set the output handler
        # output_handler = 'tabulate'
        output_handler = 'esper_output_handler'

        # register handlers
        handlers = [
            EsperOutputHandler,
            Base,
            Configure,
            Devices,
            Application,
            Command
        ]

        # hooks
        hooks = [
            ('post_setup', extend_tinydb)
        ]


class EsperTest(TestApp, Esper):
    """A sub-class of Esper that is better suited for testing."""

    class Meta:
        label = 'esper'


def main():
    with Esper() as app:
        try:
            app.run()

        except AssertionError as e:
            app.log.error('AssertionError > %s' % e.args[0])
            app.exit_code = 1

            if app.debug is True:
                import traceback
                traceback.print_exc()

        except EsperError as e:
            app.log.error('EsperError > %s' % e.args[0])
            app.exit_code = 1

            if app.debug is True:
                import traceback
                traceback.print_exc()

        except CaughtSignal as e:
            # Default Cement signals are SIGINT and SIGTERM, exit 0 (non-error)
            app.log.error('\n%s' % e)
            app.exit_code = 0


if __name__ == '__main__':
    main()
