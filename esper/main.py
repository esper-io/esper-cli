from cement import App, TestApp, init_defaults
from cement.core.exc import CaughtSignal

from esper.controllers.application.application import Application
from esper.controllers.application.version import ApplicationVersion
from esper.controllers.base import Base
from esper.controllers.configure import Configure
from esper.controllers.device.command import DeviceCommand
from esper.controllers.device.device import Device
from esper.controllers.device.group_command import GroupCommand
from esper.controllers.device.install import AppInstall
from esper.controllers.device.status import DeviceStatus
from esper.controllers.enterprise.enterprise import Enterprise
from esper.controllers.enterprise.group import EnterpriseGroup
from esper.controllers.pipeline.execute import Execution
from esper.controllers.pipeline.operation import Operation
from esper.controllers.pipeline.pipeline import Pipeline
from esper.controllers.pipeline.stage import Stage
from esper.controllers.secureadb.secureadb import SecureADB
from esper.controllers.telemetry.telemetry import Telemetry
from esper.controllers.token.token import Token
from esper.core.exc import EsperError
from esper.core.output_handler import EsperOutputHandler
from esper.ext.certs import init_certs
from esper.ext.utils import extend_tinydb

# configuration defaults
CONFIG = init_defaults('esper')
CONFIG['esper']['debug'] = False
CONFIG['esper']['creds_file'] = '~/.esper/db/creds.json'
CONFIG['esper']['certs_folder'] = '~/.esper/certs'
CONFIG['esper']['local_key'] = '~/.esper/certs/local.key'
CONFIG['esper']['local_cert'] = '~/.esper/certs/local.pem'
CONFIG['esper']['device_cert'] = '~/.esper/certs/device.pem'

# meta defaults
META = init_defaults('log.colorlog')
META['log.colorlog']['file'] = '~/.esper/logs/esper.log'
META['log.colorlog']['level'] = 'debug'
META['log.colorlog']['to_console'] = False
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
        label = 'espercli'
        debug_argument_options = ['-D', '--debug']

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
            Device,
            Application,
            DeviceCommand,
            ApplicationVersion,
            AppInstall,
            DeviceStatus,
            Enterprise,
            EnterpriseGroup,
            GroupCommand,
            SecureADB,
            Token,
            Telemetry,
            Pipeline,
            Stage,
            Operation,
            Execution
        ]

        # hooks
        hooks = [
            ('post_setup', extend_tinydb),
            ('post_setup', init_certs),
        ]


# configuration defaults for test
TEST_CONFIG = init_defaults('esper')
TEST_CONFIG['esper']['debug'] = False
TEST_CONFIG['esper']['creds_file'] = 'creds.json'
TEST_CONFIG['esper']['certs_folder'] = '~/.esper/certs'
TEST_CONFIG['esper']['local_key'] = '~/.esper/certs/local.key'
TEST_CONFIG['esper']['local_cert'] = '~/.esper/certs/local.pem'
TEST_CONFIG['esper']['device_cert'] = '~/.esper/certs/device.pem'


class EsperTest(TestApp, Esper):
    """A sub-class of Esper that is better suited for testing."""

    class Meta:
        label = 'espercli'
        config_defaults = TEST_CONFIG
        config_files = []


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
