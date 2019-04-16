
from cement import App, TestApp, init_defaults
from cement.core.exc import CaughtSignal
from .core.exc import EsperError
from .controllers.base import Base
from .controllers.configure import Configure
from .controllers.devices import Devices
from .ext.utils import extend_tinydb

# configuration defaults
CONFIG = init_defaults('esper')
CONFIG['esper']['foo'] = 'bar'
CONFIG['esper']['creds_file'] = '~/.esper/creds.json'


class Esper(App):
    """Esper CLI Tool primary application."""

    class Meta:
        label = 'esper'

        # configuration defaults
        config_defaults = CONFIG

        # call sys.exit() on close
        close_on_exit = True

        # load additional framework extensions
        extensions = [
            'yaml',
            'colorlog',
            'jinja2',
            'tabulate'
        ]

        # configuration handler
        config_handler = 'yaml'

        # configuration file suffix
        config_file_suffix = '.yml'

        # set the log handler
        log_handler = 'colorlog'

        # set the output handler
        output_handler = 'tabulate'

        # register handlers
        handlers = [
            Base,
            Configure,
            Devices
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
            print('AssertionError > %s' % e.args[0])
            app.exit_code = 1

            if app.debug is True:
                import traceback
                traceback.print_exc()

        except EsperError as e:
            print('EsperError > %s' % e.args[0])
            app.exit_code = 1

            if app.debug is True:
                import traceback
                traceback.print_exc()

        except CaughtSignal as e:
            # Default Cement signals are SIGINT and SIGTERM, exit 0 (non-error)
            print('\n%s' % e)
            app.exit_code = 0


if __name__ == '__main__':
    main()
