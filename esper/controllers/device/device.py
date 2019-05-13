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
        valid_keys = ['id', 'device_name', 'suid', 'api_level', 'template_name', 'is_gms']
        current_state = DeviceState(device.status).name

        if format == OutputFormat.JSON:
            renderable = {k: v for k, v in device.to_dict().items() if k in valid_keys}
            renderable['state'] = current_state
        else:
            title = "TITLE"
            details = "DETAILS"
            renderable = [{title: k, details: v} for k, v in device.to_dict().items() if k in valid_keys]
            renderable.append({title: 'state', details: current_state})
        return renderable

    @ex(
        help='Show device details and set as active device',
        arguments=[
            (['device_name'],
             {'help': 'Show details about the device',
              'action': 'store'}),
            (['-a', '--active'],
             {'help': 'Set device as active for further device specific commands',
              'action': 'store_true',
              'dest': 'active'}),
            (['-j', '--json'],
             {'help': 'Render result in Json format',
              'action': 'store_true',
              'dest': 'json'}),
        ]
    )
    def show(self):
        validate_creds_exists(self.app)
        db = DBWrapper(self.app.creds)
        device_name = self.app.pargs.device_name

        device_client = APIClient(db.get_configure()).get_device_api_client()
        enterprise_id = db.get_enterprise_id()

        kwargs = {'name': device_name}
        try:
            search_response = device_client.get_all_devices(enterprise_id, limit=1, offset=0, **kwargs)
            if not search_response.results or len(search_response.results) == 0:
                print(f'Device does not exist with name {device_name}')
                return
            response = search_response.results[0]
        except ApiException as e:
            self.app.log.debug(f"Failed to list devices: {e}")
            self.app.log.error(f"Failed to fetch device, reason: {e.reason}")
            return

        if self.app.pargs.active:
            db.set_device({'id': response.id, 'name': response.device_name})

        if not self.app.pargs.json:
            renderable = self._device_basic_response(response)
            self.app.render(renderable, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="fancy_grid")
        else:
            renderable = self._device_basic_response(response, OutputFormat.JSON)
            self.app.render(renderable, format=OutputFormat.JSON.value)

    @ex(
        help='Set, show or unset the active device',
        arguments=[
            (['-n', '--name'],
             {'help': 'Device name.',
              'action': 'store',
              'dest': 'name'}),
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
        device_client = APIClient(db.get_configure()).get_device_api_client()
        enterprise_id = db.get_enterprise_id()

        if self.app.pargs.name:
            device_name = self.app.pargs.name
            kwargs = {'name': device_name}
            try:
                search_response = device_client.get_all_devices(enterprise_id, limit=1, offset=0, **kwargs)
                if not search_response.results or len(search_response.results) == 0:
                    print(f'Device does not exist with name {device_name}')
                    return
                response = search_response.results[0]
                db.set_device({'id': response.id, 'name': response.device_name})
            except ApiException as e:
                self.app.log.debug(f"Failed to list devices: {e}")
                self.app.log.error(f"Failed to set active device, reason: {e.reason}")
                return
        elif self.app.pargs.unset:
            device = db.get_device()
            if device is None or device.get('name') is None:
                print('There is no active device.')
                return

            db.unset_device()
            self.app.log.debug(f"Unset the active device {device.get('name')}")
            print(f"Unset the active device {device.get('name')}")
            return
        else:
            device = db.get_device()
            if device is None or device.get('name') is None:
                print('There is no active device.')
                return

            device_id = device.get('id')
            try:
                response = device_client.get_device_by_id(enterprise_id, device_id)
            except ApiException as e:
                self.app.log.debug(f"Failed to show active device: {e}")
                self.app.log.error(f"Failed to show active device, reason: {e.reason}")
                return

        if not self.app.pargs.json:
            renderable = self._device_basic_response(response)
            self.app.render(renderable, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="fancy_grid")
        else:
            renderable = self._device_basic_response(response, OutputFormat.JSON)
            self.app.render(renderable, format=OutputFormat.JSON.value)
