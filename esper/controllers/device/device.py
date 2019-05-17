import uuid
from cement import Controller, ex
from esperclient.rest import ApiException

from esper.controllers.enums import DeviceState, OutputFormat
from esper.ext.api_client import APIClient
from esper.ext.db_wrapper import DBWrapper
from esper.ext.utils import validate_creds_exists, parse_error_message


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
             {'help': 'Group name',
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
        group_name = self.app.pargs.group
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

        if group_name:
            kw = {'name': group_name}
            group_id = None
            try:
                group_client = APIClient(db.get_configure()).get_group_api_client()
                search_response = group_client.get_all_groups(enterprise_id, limit=1, offset=0, **kw)
                for group in search_response.results:
                    if group.name == group_name:
                        group_id = group.id
                        break
            except ApiException as e:
                self.app.log.error(f"[device-list] Failed to list groups: {e}")
                self.app.render(f"ERROR: {parse_error_message(self.app, e)}")
                return

            if not group_id:
                group_id = str(uuid.uuid4())  # random uuid

            kwargs['group'] = group_id

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
            self.app.log.error(f"[device-list] Failed to list devices: {e}")
            self.app.render(f"ERROR: {parse_error_message(self.app, e)}")
            return

        self.app.render(f"Number of Devices: {response.count}")
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
            self.app.render(devices, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")
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
                self.app.log.debug(f'[device-show] Device does not exist with name {device_name}')
                self.app.render(f'Device does not exist with name {device_name}')
                return
            response = search_response.results[0]
        except ApiException as e:
            self.app.log.error(f"[device-show] Failed to list devices: {e}")
            self.app.render(f"ERROR: {parse_error_message(self.app, e)}")
            return

        if self.app.pargs.active:
            db.set_device({'id': response.id, 'name': response.device_name})

        if not self.app.pargs.json:
            renderable = self._device_basic_response(response)
            self.app.render(renderable, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")
        else:
            renderable = self._device_basic_response(response, OutputFormat.JSON)
            self.app.render(renderable, format=OutputFormat.JSON.value)

    @ex(
        help='Set, show or reset the active device',
        arguments=[
            (['-n', '--name'],
             {'help': 'Device name.',
              'action': 'store',
              'dest': 'name'}),
            (['-r', '--reset'],
             {'help': 'Reset the active device',
              'action': 'store_true',
              'dest': 'reset'}),
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
                    self.app.log.debug(f'[device-active] Device does not exist with name {device_name}')
                    self.app.render(f'Device does not exist with name {device_name}')
                    return
                response = search_response.results[0]
                db.set_device({'id': response.id, 'name': response.device_name})
            except ApiException as e:
                self.app.log.error(f"[device-active] Failed to list devices: {e}")
                self.app.render(f"ERROR: {parse_error_message(self.app, e)}")
                return
        elif self.app.pargs.reset:
            device = db.get_device()
            if device is None or device.get('name') is None:
                self.app.log.debug('[device-active] There is no active device.')
                self.app.render('There is no active device.')
                return

            db.unset_device()
            self.app.log.debug(f"[device-active] Reset the active device {device.get('name')}")
            self.app.render(f"Reset the active device {device.get('name')}")
            return
        else:
            device = db.get_device()
            if device is None or device.get('name') is None:
                self.app.log.debug('[device-active] There is no active device.')
                self.app.render('There is no active device.')
                return

            device_id = device.get('id')
            try:
                response = device_client.get_device_by_id(enterprise_id, device_id)
            except ApiException as e:
                self.app.log.error(f"[device-active] Failed to show active device: {e}")
                self.app.render(f"ERROR: {parse_error_message(self.app, e)}")
                return

        if not self.app.pargs.json:
            renderable = self._device_basic_response(response)
            self.app.render(renderable, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")
        else:
            renderable = self._device_basic_response(response, OutputFormat.JSON)
            self.app.render(renderable, format=OutputFormat.JSON.value)
