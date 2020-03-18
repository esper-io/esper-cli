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

    @staticmethod
    def get_name_and_tags_from_device(device):
        """
        Returns device name and tags
        :param device: Device objects response
        :return: name, tags
        """
        return device.alias_name if device.alias_name else device.device_name, \
               ', '.join(device.tags) if device.tags and len(device.tags) > 0 else ''

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
            (['-se', '--serial'],
             {'help': 'Serial number',
              'action': 'store',
              'dest': 'serial'}),
            (['-t', '--tags'],
             {'help': 'Tags',
              'action': 'store',
              'dest': 'tags'}),
            (['--search'],
             {'help': 'Search for device name, alias_name or device id',
              'action': 'store',
              'dest': 'search'}),
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
        serial = self.app.pargs.serial
        tags = self.app.pargs.tags
        search = self.app.pargs.search
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
                self.app.render(f"ERROR: {parse_error_message(self.app, e)}\n")
                return

            if not group_id:
                group_id = str(uuid.uuid4())  # random uuid

            kwargs['group'] = group_id

        if imei:
            kwargs['imei'] = imei

        if serial:
            kwargs['serial'] = serial

        if search:
            kwargs['search'] = search

        if tags:
            kwargs['tags'] = tags

        if brand:
            kwargs['brand'] = brand

        if gms:
            kwargs['is_gms'] = gms

        try:
            # Find devices in an enterprise
            response = device_client.get_all_devices(enterprise_id, limit=limit, offset=offset, **kwargs)
        except ApiException as e:
            self.app.log.error(f"[device-list] Failed to list devices: {e}")
            self.app.render(f"ERROR: {parse_error_message(self.app, e)}\n")
            return

        self.app.render(f"Number of Devices: {response.count}")
        if not self.app.pargs.json:
            devices = []

            label = {
                'id': "ID",
                'name': "NAME",
                'model': "MODEL",
                'state': "CURRENT STATE",
                'tags': "TAGS"
            }

            for device in response.results:
                current_state = DeviceState(device.status).name
                name, tags = self.get_name_and_tags_from_device(device)

                devices.append(
                    {
                        label['id']: device.id,
                        label['name']: name,
                        label['model']: device.hardware_info.get("manufacturer"),
                        label['state']: current_state,
                        label['tags']: tags
                    }
                )
            self.app.render(devices, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")
        else:
            devices = []
            for device in response.results:
                current_state = DeviceState(device.status).name
                name, _ = self.get_name_and_tags_from_device(device)
                devices.append(
                    {
                        'id': device.id,
                        'device': name,
                        'model': device.hardware_info.get("manufacturer"),
                        'state': current_state,
                        'tags': device.tags
                    }
                )
            self.app.render(devices, format=OutputFormat.JSON.value)

    def _device_basic_response(self, device, format=OutputFormat.TABULATED):
        valid_keys = ['id', 'device_name', 'alias_name', 'suid', 'api_level', 'template_name', 'is_gms']
        current_state = DeviceState(device.status).name

        if format == OutputFormat.JSON:
            renderable = {k: v for k, v in device.to_dict().items() if k in valid_keys}
            renderable['state'] = current_state
            renderable['tags'] = device.tags
        else:
            title = "TITLE"
            details = "DETAILS"
            renderable = [{title: k, details: v} for k, v in device.to_dict().items() if k in valid_keys]
            renderable.append({title: 'state', details: current_state})
            _, tags = self.get_name_and_tags_from_device(device)
            renderable.append({title: 'tags', details: tags})
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
                self.app.render(f'Device does not exist with name {device_name}\n')
                return
            response = search_response.results[0]
        except ApiException as e:
            self.app.log.error(f"[device-show] Failed to list devices: {e}")
            self.app.render(f"ERROR: {parse_error_message(self.app, e)}\n")
            return

        if self.app.pargs.active:
            name, _ = self.get_name_and_tags_from_device(response)
            db.set_device({'id': response.id, 'name': name})

        if not self.app.pargs.json:
            renderable = self._device_basic_response(response)
            self.app.render(renderable, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")
        else:
            renderable = self._device_basic_response(response, OutputFormat.JSON)
            self.app.render(renderable, format=OutputFormat.JSON.value)

    @ex(
        help='Set or show the active device',
        arguments=[
            (['-n', '--name'],
             {'help': 'Device name.',
              'action': 'store',
              'dest': 'name'}),
            (['-j', '--json'],
             {'help': 'Render result in Json format',
              'action': 'store_true',
              'dest': 'json'}),
        ]
    )
    def set_active(self):
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
                    self.app.render(f'Device does not exist with name {device_name}\n')
                    return
                response = search_response.results[0]
                name, _ = self.get_name_and_tags_from_device(response)
                db.set_device({'id': response.id, 'name': name})
            except ApiException as e:
                self.app.log.error(f"[device-active] Failed to list devices: {e}")
                self.app.render(f"ERROR: {parse_error_message(self.app, e)}\n")
                return
        else:
            device = db.get_device()
            if device is None or device.get('name') is None:
                self.app.log.debug('[device-active] There is no active device.')
                self.app.render('There is no active device.\n')
                return

            device_id = device.get('id')
            try:
                response = device_client.get_device_by_id(enterprise_id, device_id)
            except ApiException as e:
                self.app.log.error(f"[device-active] Failed to show active device: {e}")
                self.app.render(f"ERROR: {parse_error_message(self.app, e)}\n")
                return

        if not self.app.pargs.json:
            renderable = self._device_basic_response(response)
            self.app.render(renderable, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")
        else:
            renderable = self._device_basic_response(response, OutputFormat.JSON)
            self.app.render(renderable, format=OutputFormat.JSON.value)

    @ex(
        help='Unset the active device',
        arguments=[]
    )
    def unset_active(self):
        validate_creds_exists(self.app)
        db = DBWrapper(self.app.creds)

        device = db.get_device()
        if device is None or device.get('name') is None:
            self.app.log.debug('[device-active] There is no active device.')
            self.app.render('There is no active device.\n')
            return

        db.unset_device()
        self.app.log.debug(f"[device-active] Unset the active device {device.get('name')}")
        self.app.render(f"Unset the active device {device.get('name')}\n")
