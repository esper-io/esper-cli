from cement import Controller, ex
from cement.utils.version import get_version_banner
from crayons import white
from esperclient.rest import ApiException

from esper.controllers.enums import OutputFormat
from esper.core.version import get_version
from esper.ext.api_client import APIClient
from esper.ext.db_wrapper import DBWrapper
from esper.ext.utils import validate_creds_exists

VERSION_BANNER = """
Command Line Tool for Esper SDK %s
%s
""" % (get_version(), get_version_banner())


class AppInstall(Controller):
    class Meta:
        label = 'install'

        # text displayed at the top of --help output
        description = 'Application Installs commands'

        # text displayed at the bottom of --help output
        epilog = 'Usage: espercli install'

        stacked_type = 'nested'
        stacked_on = 'base'

    @ex(
        help='List application installs',
        arguments=[
            (['-d', '--device'],
             {'help': 'Device id',
              'action': 'store',
              'dest': 'device'}),
            (['-an', '--appname'],
             {'help': 'Application name',
              'action': 'store',
              'dest': 'appname'}),
            (['-p', '--package'],
             {'help': 'Package name',
              'action': 'store',
              'dest': 'package'}),
            (['-s', '--state'],
             {'help': 'Install state',
              'action': 'store',
              'dest': 'state'}),
            (['-l', '--limit'],
             {'help': 'Number of results to return per page',
              'action': 'store',
              'default': 20,
              'dest': 'limit'}),
            (['-i', '--offset'],
             {'help': 'The initial index from which to return the results',
              'action': 'store',
              'default': 0,
              'dest': 'offset'}),
            (['-j', '--json'],
             {'help': 'Render result in Json format',
              'action': 'store_true',
              'dest': 'json'}),
        ]
    )
    def list(self):
        """Command to list installs"""
        validate_creds_exists(self.app)
        db = DBWrapper(self.app.creds)
        device_client = APIClient(db.get_configure()).get_device_api_client()
        enterprise_id = db.get_enterprise_id()

        if self.app.pargs.device:
            device_id = self.app.pargs.device
        else:
            device = db.get_device()
            if not device or not device.get('id'):
                self.app.log.info('Not set the active device.')
                return

            device_id = device.get('id')

        app_name = self.app.pargs.appname
        package_name = self.app.pargs.package
        install_state = self.app.pargs.state
        limit = self.app.pargs.limit
        offset = self.app.pargs.offset

        kwargs = {}
        if app_name:
            kwargs['application_name'] = app_name

        if package_name:
            kwargs['package_name'] = package_name

        if install_state:
            kwargs['install_state'] = install_state

        try:
            response = device_client.get_app_installs(enterprise_id, device_id, limit=limit, offset=offset, **kwargs)
        except ApiException as e:
            self.app.log.debug(f"Failed to list installs: {e}")
            self.app.log.error(f"Failed to list installs, reason: {e.reason}")
            return

        if not self.app.pargs.json:
            installs = []

            label = {
                'id': white("ID", bold=True),
                'application_name': white("APPLICATION", bold=True),
                'package_name': white("PACKAGE", bold=True),
                'version_code': white("VERSION", bold=True),
                'install_state': white("STATE", bold=True),
                'reason_details': white("REASON", bold=True)
            }

            for install in response.results:
                installs.append(
                    {
                        label['id']: install.id,
                        label['application_name']: install.application.application_name,
                        label['package_name']: install.application.package_name,
                        label['version_code']: install.application.version.version_code,
                        label['install_state']: install.install_state,
                        label['reason_details']: install.reason_details
                    }
                )
            print(white(f"\tTotal Number of Installs: {response.count}", bold=True))
            self.app.render(installs, format=OutputFormat.TABULATED.value, headers="keys",
                            tablefmt="fancy_grid")
        else:
            installs = []
            for install in response.results:
                installs.append(
                    {
                        'id': install.id,
                        'application_name': install.application.application_name,
                        'package_name': install.application.package_name,
                        'version_code': install.application.version.version_code,
                        'install_state': install.install_state,
                        'reason_details': install.reason_details
                    }
                )
            print(white(f"Total Number of Installs: {response.count}", bold=True))
            self.app.render(installs, format=OutputFormat.JSON.value)
