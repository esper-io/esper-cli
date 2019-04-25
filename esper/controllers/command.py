from cement import Controller, ex
from cement.utils.version import get_version_banner
from crayons import white
from esperclient.rest import ApiException

from .enums import OutputFormat, DeviceCommandEnum
from ..core.version import get_version
from ..ext.utils import validate_creds_exists, get_client_config, get_command_api, get_command_request

VERSION_BANNER = """
Command Line Tool for Esper SDK %s
%s
""" % (get_version(), get_version_banner())


class Command(Controller):
    class Meta:
        label = 'command'

        # text displayed at the top of --help output
        # description = 'Applications command to list and show application details'

        # text displayed at the bottom of --help output
        # epilog = 'Usage: espercli applications'

        stacked_type = 'embedded'
        stacked_on = 'base'

    @ex(
        help='Command to install the application version',
        arguments=[
            (['version_id'],
             {'help': 'Application version id',
              'action': 'store'}),
            (['-d', '--device'],
             {'help': 'Device id which to fire command',
              'action': 'store',
              'dest': 'device'}),
            (['-j', '--json'],
             {'help': 'Render result in Json format',
              'action': 'store_true',
              'dest': 'json'}),
        ]
    )
    def install(self):
        """Command to install application"""
        try:

            validate_creds_exists(self.app)
            config = get_client_config(self.app)
            api_instance = get_command_api(config)
            enterprise_id = self.app.creds.all()[0].get("enterprise")

            version_id = self.app.pargs.version_id
            device_id = self.app.pargs.device

            if not device_id:
                self.app.log.error("Device id is required to install an application version")

            command_request = get_command_request(command_name=DeviceCommandEnum.INSTALL.name,
                                                  command_args={"app_version": version_id})
            response = api_instance.run_command(enterprise_id, device_id, command_request)

            valid_keys = ['id', 'command', 'state']
            print(white(f"\tCOMMAND DETAILS of {response.command}", bold=True))

            if not self.app.pargs.json:
                title = white("TITLE", bold=True)
                details = white("DETAILS", bold=True)
                renderable = [{title: k, details: v} for k, v in response.to_dict().items() if k in valid_keys]
                self.app.render(renderable, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="fancy_grid")
            else:
                renderable = {k: v for k, v in response.to_dict().items() if k in valid_keys}
                self.app.render(renderable, format=OutputFormat.JSON.value)

        except ApiException as e:
            self.app.log.debug(f"Failed to fire the install command: {e}")
            self.app.log.error(f"Failed to fire the install command, reason: {e.reason}")
