from cement import Controller, ex
from cement.utils.version import get_version_banner
from crayons import white, red, green, blue
from esperclient.rest import ApiException

from .enums import DeviceState, OutputFormat
from ..core.version import get_version
from ..ext.utils import validate_creds_exists, get_client_config, get_device_api_instance

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

    @ex(
        help='Devices command to list device details',
        arguments=[
            (['--filter'],
             {'help': 'Filter devices by device state',
              'action': 'store',
              'choices': ['active', 'inactive', 'disabled'],
              'dest': 'filter_state'}),
            (['--json'],
             {'help': 'Render result in Json format',
              'action': 'store_true',
              'dest': 'json'}),
        ]
    )
    def devices(self):
        """Command to list devices"""
        try:

            validate_creds_exists(self.app)
            config = get_client_config(self.app)
            api_instance = get_device_api_instance(config)
            enterprise_id = self.app.creds.all()[0].get("enterprise")

            state = self.app.pargs.filter_state

            # Find devices in an enterprise
            response = api_instance.get_all_devices(enterprise_id)
            print(white(f"Number of Devices: {response.count}", bold=True))

            if not self.app.pargs.json:
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

                self.app.render(devices, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="fancy_grid")
            else:
                devices = []
                for device in response.results:
                    if device.status == DeviceState.ACTIVE.value:
                        current_state = DeviceState.ACTIVE.name
                    elif device.status == DeviceState.INACTIVE.value:
                        current_state = DeviceState.INACTIVE.name
                    elif device.status == DeviceState.DISABLED.value:
                        current_state = DeviceState.DISABLED.name
                    elif device.status == DeviceState.POLICY_APPLICATION_IN_PROGRESS.value:
                        current_state = DeviceState.POLICY_APPLICATION_IN_PROGRESS.name
                    else:
                        current_state = DeviceState(device.status).name
                    devices.append({'device': device.device_name,
                                    'model': device.hardware_info.get("manufacturer"),
                                    'state': current_state})
                self.app.render(devices, format=OutputFormat.JSON.value)

        except ApiException as e:
            self.app.log.error("Exception when calling DeviceApi->get_all_devices: %s\n" % e)

    @ex(
        help='Show device-specific details',
        arguments=[
            (['device_id'],
             {'help': 'Show details about the device',
              'action': 'store'}),
            (['--json'],
             {'help': 'Render result in Json format',
              'action': 'store_true',
              'dest': 'json'}),
        ]
    )
    def device(self):
        try:
            device_id = self.app.pargs.device_id

            validate_creds_exists(self.app)
            config = get_client_config(self.app)
            api_instance = get_device_api_instance(config)
            enterprise_id = self.app.creds.all()[0].get("enterprise")

            response = api_instance.get_device_by_id(enterprise_id, device_id)

            print(white(f"\tDEVICE DETAILS of {response.device_name}", bold=True))

            valid_keys = ['device_name', 'status', 'state', 'suid', 'api_level', 'template_name', 'is_gms']

            if not self.app.pargs.json:
                title = white("TITLE", bold=True)
                details = white("DETAILS", bold=True)
                renderable = [{title: k, details: v} for k, v in response.to_dict().items() if k in valid_keys]
                self.app.render(renderable, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="fancy_grid")
            else:
                renderable = {k: v for k, v in response.to_dict().items() if k in valid_keys}
                self.app.render(renderable, format=OutputFormat.JSON.value)

        except ApiException:
            self.app.log.error("Error when calling DeviceApi->get_device_by_id!")
