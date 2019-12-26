from cement import Controller, ex
from esperclient.rest import ApiException

from esper.controllers.enums import OutputFormat
from esper.ext.api_client import APIClient
from esper.ext.db_wrapper import DBWrapper
from esper.ext.utils import validate_creds_exists, parse_error_message


class AppInstall(Controller):
    class Meta:
        label = 'installs'

        # text displayed at the top of --help output
        description = 'Application Installs commands'

        # text displayed at the bottom of --help output
        epilog = 'Usage: espercli installs'

        stacked_type = 'nested'
        stacked_on = 'base'

    @ex(
        help='List application installs',
        arguments=[
            (['-d', '--device'],
             {'help': 'Device name',
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
            device_name = self.app.pargs.device
            kwargs = {'name': device_name}
            try:
                search_response = device_client.get_all_devices(enterprise_id, limit=1, offset=0, **kwargs)
                if not search_response.results or len(search_response.results) == 0:
                    self.app.log.debug(f'[installs-list] Device does not exist with name {device_name}')
                    self.app.render(f'Device does not exist with name {device_name}\n')
                    return
                response = search_response.results[0]
                device_id = response.id
            except ApiException as e:
                self.app.log.error(f"[installs-list] Failed to list devices: {e}")
                self.app.render(f"ERROR: {parse_error_message(self.app, e)}\n")
                return
        else:
            device = db.get_device()
            if not device or not device.get('id'):
                self.app.log.debug('[installs-list] There is no active device.')
                self.app.render('There is no active device.\n')
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
            self.app.log.error(f"[installs-list] Failed to list installs: {e}")
            self.app.render(f"ERROR: {parse_error_message(self.app, e)}\n")
            return

        self.app.render(f"Total Number of Installs: {response.count}")
        if not self.app.pargs.json:
            installs = []

            label = {
                'id': "ID",
                'application_name': "APPLICATION",
                'package_name': "PACKAGE",
                'version_code': "VERSION",
                'install_state': "STATE"
            }

            for install in response.results:
                installs.append(
                    {
                        label['id']: install.id,
                        label['application_name']: install.application.application_name,
                        label['package_name']: install.application.package_name,
                        label['version_code']: install.application.version.version_code,
                        label['install_state']: install.install_state
                    }
                )
            self.app.render(installs, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")
        else:
            installs = []
            for install in response.results:
                installs.append(
                    {
                        'id': install.id,
                        'application_name': install.application.application_name,
                        'package_name': install.application.package_name,
                        'version_code': install.application.version.version_code,
                        'install_state': install.install_state
                    }
                )
            self.app.render(installs, format=OutputFormat.JSON.value)
