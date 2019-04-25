from cement import Controller, ex
from cement.utils.version import get_version_banner
from clint.textui import prompt, validators

from esper.controllers.enums import OutputFormat
from ..core.version import get_version
from esper.ext.utils import validate_creds_exists

VERSION_BANNER = """
Esper CLI tool to manage resources on Esper.io API service %s
%s
""" % (get_version(), get_version_banner())


class Configure(Controller):
    class Meta:
        label = 'configure'
        stacked_type = 'embedded'
        stacked_on = 'base'

        # text displayed at the top of --help output
        description = 'Esper CLI tool to manage resources on Esper.io API service'

        # text displayed at the bottom of --help output
        epilog = 'Usage: espercli configure'

        # controller level arguments. ex: 'espercli --version'
        arguments = [
            # ### add a version banner
            # ( [ '-v', '--version' ],
            #   { 'action'  : 'version',
            #     'version' : VERSION_BANNER } ),
        ]

    @ex(
        help='Configure the credentials for `esper.io` API Service',

        # sub-command level arguments. ex: 'espercli configure --foo bar'
        arguments=[
            (['-s', '--set'],
             {'help': 'Create or Update credentials for Esper.io API Service',
              'action': 'store_true',
              'default': False,
              'dest': 'set'}),

            (['-l', '--list'],
             {'help': 'List credentials for Esper.io API Service',
              'action': 'store_true',
              'default': False,
              'dest': 'list'}),

            (['--json'],
             {'help': 'Render result in Json format',
              'action': 'store_true',
              'dest': 'json'}),
        ],
    )
    def configure(self):
        """Configure the credentials and host endpoints of the customer"""

        # Trigger the Insert operation, if --set is given OR if the Creds DB is empty
        if self.app.pargs.set or len(self.app.creds.all()) == 0:
            user = prompt.query("Enter your Username : ")
            password = prompt.query("Enter your Password : ")
            host = prompt.query("Enter your Host Endpoint:",
                                default="demo",
                                validators=[])

            enterprise = prompt.query("Enter your Enterprise Id: ",
                                      validators=[validators.RegexValidator(
                                          regex='[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}')])

            creds = [
                ("username", user),
                ("password", password),
                ("enterprise", enterprise),
                ("host", host)
            ]

            # Clear the DB
            self.app.log.debug("Purging Credentials DB...")
            self.app.creds.purge()

            # Insert the New contents into DB
            self.app.log.debug("Inserting new Credentials to DB...")
            self.app.creds.insert(dict(creds))

        # Trigger listing operation, if --list is given or Creds DB has content
        if self.app.pargs.list or len(self.app.creds.all()) == 1:
            validate_creds_exists(self.app)

            if not self.app.pargs.json:
                creds = [(k, v) for k, v in self.app.creds.all()[0].items()]
                # Render the Credentials
                self.app.render(creds, format=OutputFormat.TABULATED.value, headers=["Title", "Details"],
                                tablefmt="fancy_grid")
            else:
                self.app.render(self.app.creds.all()[0], format=OutputFormat.JSON.value)
