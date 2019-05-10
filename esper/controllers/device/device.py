from cement import Controller, ex
from cement.utils.version import get_version_banner
from esperclient.rest import ApiException

from esper.controllers.enums import DeviceState, OutputFormat
from esper.core.version import get_version
from esper.ext.api_client import APIClient
from esper.ext.db_wrapper import DBWrapper
from esper.ext.utils import validate_creds_exists

VERSION_BANNER = """
Command Line Tool for Esper SDK %s
%s
""" % (get_version(), get_version_banner())


class Device(Controller):
    class Meta:
        label = 'device'

        # text displayed at the top of --help output
        description = 'Device commands'

        # text displayed at the bottom of --help output
        epilog = 'Usage: espercli device'

        stacked_type = 'nested'
        stacked_on = 'base'

    @ex(
        help='List devices',
        arguments=[
            (['-s', '--state'],
             {'help': 'Device state',
              'action': 'store',
              'choices': ['active', 'inactive', 'disabled'],
              'dest': 'state'}),
            (['-n', '--name'],
             {'help': 'Device name',
              'action': 'store',
              'dest': 'name'}),
            (['-g', '--group'],
             {'help': 'Group id',
              'action': 'store',
              'dest': 'group'}),
            (['-im', '--imei'],
             {'help': 'IMEI number',
              'action': 'store',
              'dest': 'imei'}),
            (['-b', '--brand'],
             {'help': 'Brand name',
              'action': 'store',
              'dest': 'brand'}),
            (['-gm', '--gms'],
             {'help': 'GMS or not',
              'action': 'store',
              'choices': ['true', 'false'],
              'dest': 'gms'}),
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
        """Command to list devices"""
        validate_creds_exists(self.app)
        db = DBWrapper(self.app.creds)
        device_client = APIClient(db.get_configure()).get_device_api_client()
        enterprise_id = db.get_enterprise_id()

        state = self.app.pargs.state
        name = self.app.pargs.name
        group = self.app.pargs.group
        imei = self.app.pargs.imei
        brand = self.app.pargs.brand
        gms = self.app.pargs.gms
        limit = self.app.pargs.limit
        offset = self.app.pargs.offset

        kwargs = {}
        if state:
            kwargs['state'] = DeviceState[state.upper()].value

        if name:
            kwargs['name'] = name

        if group:
            kwargs['group'] = group

        if imei:
            kwargs['imei'] = imei

        if brand:
            kwargs['brand'] = brand

        if gms:
            kwargs['is_gms'] = gms

        try:
            # Find devices in an enterprise
            response = device_client.get_all_devices(enterprise_id, limit=limit, offset=offset, **kwargs)
        except ApiException as e:
            self.app.log.debug(f"Failed to list devices: {e}")
            self.app.log.error(f"Failed to list devices, reason: {e.reason}")
            return

        if not self.app.pargs.json:
            devices = []

            label = {
                'id': "ID",
                'name': "NAME",
                'model': "MODEL",
                'state': "CURRENT STATE"
            }

            for device in response.results:
                current_state = DeviceState(device.status).name
                devices.append(
                    {
                        label['id']: device.id,
                        label['name']: device.device_name,
                        label['model']: device.hardware_info.get("manufacturer"),
                        label['state']: current_state
                    }
                )
            print(f"Number of Devices: {response.count}")
            self.app.render(devices, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="fancy_grid")
        else:
            devices = []
            for device in response.results:
                current_state = DeviceState(device.status).name
                devices.append(
                    {
                        'id': device.id,
                        'device': device.device_name,
                        'model': device.hardware_info.get("manufacturer"),
                        'state': current_state
                    }
                )
            print(f"Number of Devices: {response.count}")
            self.app.render(devices, format=OutputFormat.JSON.value)

    def _device_basic_response(self, device, format=OutputFormat.TABULATED):
        valid_keys = ['id', 'device_name', 'status', 'state', 'suid', 'api_level', 'template_name', 'is_gms']

        if format == OutputFormat.JSON:
            renderable = {k: v for k, v in device.to_dict().items() if k in valid_keys}
        else:
            title = "TITLE"
            details = "DETAILS"
            renderable = [{title: k, details: v} for k, v in device.to_dict().items() if k in valid_keys]
        return renderable

    @ex(
        help='Show device details and set as active device',
        arguments=[
            (['device_id'],
             {'help': 'Show details about the device',
              'action': 'store'}),
            (['-s', '--set'],
             {'help': 'Set device as active for further device specific commands',
              'action': 'store_true',
              'dest': 'set'}),
            (['-j', '--json'],
             {'help': 'Render result in Json format',
              'action': 'store_true',
              'dest': 'json'}),
        ]
    )
    def show(self):
        validate_creds_exists(self.app)
        db = DBWrapper(self.app.creds)
        device_id = self.app.pargs.device_id

        if self.app.pargs.set:
            db.set_device({'id': device_id})

        device_client = APIClient(db.get_configure()).get_device_api_client()
        enterprise_id = db.get_enterprise_id()

        try:
            response = device_client.get_device_by_id(enterprise_id, device_id)
        except ApiException as e:
            self.app.log.debug(f"Failed to show details of a device: {e}")
            self.app.log.error(f"Failed to show details of a device, reason: {e.reason}")
            return

        print(f"DEVICE DETAILS of {response.device_name}")
        if not self.app.pargs.json:
            renderable = self._device_basic_response(response)
            self.app.render(renderable, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="fancy_grid")
        else:
            renderable = self._device_basic_response(response, OutputFormat.JSON)
            self.app.render(renderable, format=OutputFormat.JSON.value)

    @ex(
        help='Show or unset the active device',
        arguments=[
            (['-u', '--unset'],
             {'help': 'Unset the active device',
              'action': 'store_true',
              'dest': 'unset'}),
            (['-j', '--json'],
             {'help': 'Render result in Json format',
              'action': 'store_true',
              'dest': 'json'}),
        ]
    )
    def active(self):
        validate_creds_exists(self.app)
        db = DBWrapper(self.app.creds)

        device_id = None
        device = db.get_device()
        if device:
            device_id = device.get('id', None)

        if self.app.pargs.unset:
            if not device_id:
                self.app.log.info('Not set the active device.')
                return

            db.unset_device()
            self.app.log.info(f'Unset the active device {device_id}')
            return

        if not device_id:
            self.app.log.info("Not set the active device.")
            return

        device_client = APIClient(db.get_configure()).get_device_api_client()
        enterprise_id = db.get_enterprise_id()

        try:
            response = device_client.get_device_by_id(enterprise_id, device_id)
        except ApiException as e:
            self.app.log.debug(f"Failed to show or unset the active device: {e}")
            self.app.log.error(f"Failed to show or unset the active device, reason: {e.reason}")
            return

        print(f"DEVICE DETAILS of {response.device_name}")
        if not self.app.pargs.json:
            renderable = self._device_basic_response(response)
            self.app.render(renderable, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="fancy_grid")
        else:
            renderable = self._device_basic_response(response, OutputFormat.JSON)
            self.app.render(renderable, format=OutputFormat.JSON.value)
