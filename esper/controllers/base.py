
from cement import Controller, ex
from cement.utils.version import get_version_banner
from ..core.version import get_version

VERSION_BANNER = """
Esper CLI tool to manage resources on Esper.io API service:  %s
%s
""" % (get_version(), get_version_banner())


class Base(Controller):
    class Meta:
        label = 'base'

        # text displayed at the top of --help output
        description = 'Esper CLI tool to manage resources on Esper.io API service'

        # text displayed at the bottom of --help output
        epilog = 'Usage: esper <sub-command> [--options]'

        # controller level arguments. ex: 'esper --version'
        arguments = [
            ### add a version banner
            ( [ '-v', '--version' ],
              { 'action'  : 'version',
                'version' : VERSION_BANNER } ),
        ]

    def _default(self):
        """Default action if no sub-command is passed."""

        self.app.args.print_help()
    #
    #
    # @ex(
    #     help='example sub command1',
    #
    #     # sub-command level arguments. ex: 'esper command1 --foo bar'
    #     arguments=[
    #         ### add a sample foo option under subcommand namespace
    #         ( [ '-f', '--foo' ],
    #           { 'help' : 'notorious foo option',
    #             'action'  : 'store',
    #             'dest' : 'foo' } ),
    #     ],
    # )
    # def command1(self):
    #     """Example sub-command."""
    #
    #     data = {
    #         'foo' : 'bar',
    #     }
    #
    #     ### do something with arguments
    #     if self.app.pargs.foo is not None:
    #         data['foo'] = self.app.pargs.foo
    #
    #     self.app.render(data, 'command1.jinja2')
