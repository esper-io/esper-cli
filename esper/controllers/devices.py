import sys

from cement import Controller, ex
from cement.utils.version import get_version_banner
from crayons import white, red, green, blue
from esperclient.rest import ApiException

from .enums import DeviceState
from ..core.version import get_version
from ..ext.utils import get_client_config, get_device_api_instance

VERSION_BANNER = """
Command Line Tool for Esper SDK %s
%s
""" % (get_version(), get_version_banner())


class Devices(Controller):

    class Meta:
        label = 'devices'

        # text displayed at the top of --help output
        # description = 'Devices command to list and show device details'

        # text displayed at the bottom of --help output
        # epilog = 'Usage: esperapp devices'

        stacked_type = 'embedded'
        stacked_on = 'base'

    def validate_creds_exists(self):
        if len(self.app.creds.all()) == 0:
            print(red("Credentials have not been set!", bold=True))
            print(white("Setup credentials by calling `configure` command."))

            sys.exit(1)

    @ex(
        help='Devices command to list device details',
        arguments=[
            (['--filter'],
             {'help': 'Filter devices by device state',
              'action': 'store',
              'choices': ['active', 'inactive', 'disabled'],
              'dest': 'filter_state'}),
        ]
    )
    def devices(self):
        """Command to list devices"""
        try:

            self.validate_creds_exists()

            config = get_client_config(self.app)
            api_instance = get_device_api_instance(config)
            enterprise_id = self.app.creds.all()[0].get("enterprise")

            state = self.app.pargs.filter_state

            # Find devices in an enterprise
            response = api_instance.get_all_devices(enterprise_id)

            devices = []

            label = {
                'device': white("DEVICE", bold=True),
                'model': white("MODEL", bold=True),
                'state': white("CURRENT STATE", bold=True)
            }

            for device in response.results:
                if state and device.status != DeviceState[state.upper()].value:
                    continue

                if device.status == DeviceState.ACTIVE.value:
                    current_state = blue("Active")
                elif device.status == DeviceState.INACTIVE.value:
                    current_state = red("Inactive")
                elif device.status == DeviceState.DISABLED.value:
                    current_state = white("Disabled")
                elif device.status == DeviceState.POLICY_APPLICATION_IN_PROGRESS.value:
                    current_state = green("Applying policy...")
                else:
                    current_state = DeviceState(device.status).name

                devices.append({label['device']: device.device_name,
                                label['model']: device.hardware_info.get("manufacturer"),
                                label['state']: current_state})

            print(white(f"Number of Devices: {response.count}", bold=True))

            self.app.render(devices, headers="keys", tablefmt="fancy_grid")

            # self.app.render({"count": api_response.count, "devices": devices}, template="device_list.jinja2")

        except ApiException as e:
            print(red("Exception when calling DeviceApi->get_all_devices: %s\n" % e, bold=True))

    @ex(
        help='Show device-specific details',
        arguments=[
            (['device_id'],
             {'help': 'Show details about the device',
              'action': 'store'}),
        ]
    )
    def device(self):
        try:
            device_id = self.app.pargs.device_id

            response = self.api_instance.get_device_by_id(self.enterprise_id, device_id)

            print(white(f"\tDEVICE DETAILS of {response.device_name}", bold=True))

            valid_keys = ['device_name', 'status', 'state', 'suid', 'api_level', 'template_name', 'is_gms']

            title = white("TITLE", bold=True)
            details = white("DETAILS", bold=True)
            renderable = [{title: k, details: v} for k, v in response.to_dict().items() if k in valid_keys]
            self.app.render(renderable, headers="keys", tablefmt="fancy_grid")

        except ApiException:
            print(red("Error when calling DeviceApi->get_device_by_id!", bold=True))
