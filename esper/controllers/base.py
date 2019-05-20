from cement import Controller
from cement.utils.version import get_version_banner

from ..core.version import get_version

VERSION_BANNER = """
Esper CLI is Command line tool for the Esper APIs:  %s
%s
""" % (get_version(), get_version_banner())


class Base(Controller):
    class Meta:
        label = 'base'

        # text displayed at the top of --help output
        description = 'Esper CLI is Command line tool for the Esper APIs'

        # text displayed at the bottom of --help output
        epilog = 'Usage: espercli <sub-command> [--options]'

        # controller level arguments. ex: 'esper --version'
        arguments = [
            ### add a version banner
            (['-v', '--version'],
             {'action': 'version',
              'version': VERSION_BANNER}),
        ]

    def _default(self):
        """Default action if no sub-command is passed."""

        self.app.args.print_help()
