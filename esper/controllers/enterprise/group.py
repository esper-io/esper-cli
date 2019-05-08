from cement import Controller, ex
from cement.utils.version import get_version_banner
from crayons import white
from esperclient import DeviceGroup, DeviceGroupUpdate
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


class EnterpriseGroup(Controller):
    class Meta:
        label = 'group'

        # text displayed at the top of --help output
        description = 'Device group commands'

        # text displayed at the bottom of --help output
        epilog = 'Usage: espercli group'

        stacked_type = 'nested'
        stacked_on = 'base'

    @ex(
        help='List groups',
        arguments=[
            (['-n', '--name'],
             {'help': 'Filter groups by group name',
              'action': 'store',
              'dest': 'name'}),
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
              'dest': 'json'})
        ]
    )
    def list(self):
        validate_creds_exists(self.app)
        db = DBWrapper(self.app.creds)
        group_client = APIClient(db.get_configure()).get_group_api_client()
        enterprise_id = db.get_enterprise_id()

        name = self.app.pargs.name
        limit = self.app.pargs.limit
        offset = self.app.pargs.offset

        kwargs = {}
        if name:
            kwargs['name'] = name

        try:
            response = group_client.get_all_groups(enterprise_id, limit=limit, offset=offset, **kwargs)
        except ApiException as e:
            self.app.log.debug(f"Failed to list groups: {e}")
            self.app.log.error(f"Failed to list groups, reason: {e.reason}")
            return

        if not self.app.pargs.json:
            groups = []

            label = {
                'id': white("ID", bold=True),
                'name': white("NAME", bold=True),
                'device_count': white("DEVICE COUNT", bold=True)
            }

            for group in response.results:
                groups.append(
                    {
                        label['id']: group.id,
                        label['name']: group.name,
                        label['device_count']: group.device_count if group.device_count else 0
                    }
                )
            print(white(f"\tTotal Number of Groups: {response.count}", bold=True))
            self.app.render(groups, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="fancy_grid")
        else:
            groups = []
            for group in response.results:
                groups.append(
                    {
                        'id': group.id,
                        'name': group.name,
                        'device_count': group.device_count if group.device_count else 0
                    }
                )
            print(white(f"Total Number of Groups: {response.count}", bold=True))
            self.app.render(groups, format=OutputFormat.JSON.value)

    def _group_basic_response(self, group, format=OutputFormat.TABULATED):
        if format == OutputFormat.TABULATED:
            title = white("TITLE", bold=True)
            details = white("DETAILS", bold=True)
            renderable = [
                {title: 'id', details: group.id},
                {title: 'name', details: group.name},
                {title: 'device_count', details: group.device_count}
            ]
        else:
            renderable = {
                'id': group.id,
                'name': group.name,
                'device_count': group.device_count
            }

        return renderable

    @ex(
        help='Show group details and set as active group',
        arguments=[
            (['group_id'],
             {'help': 'Group id',
              'action': 'store'}),
            (['-s', '--set'],
             {'help': 'Set this group as active group for further group specific commands',
              'action': 'store_true',
              'dest': 'set'}),
            (['-j', '--json'],
             {'help': 'Render result in Json format',
              'action': 'store_true',
              'dest': 'json'})
        ]
    )
    def show(self):
        validate_creds_exists(self.app)
        db = DBWrapper(self.app.creds)
        group_client = APIClient(db.get_configure()).get_group_api_client()
        enterprise_id = db.get_enterprise_id()

        group_id = self.app.pargs.group_id

        if self.app.pargs.set:
            db.set_group({'id': group_id})

        try:
            response = group_client.get_group_by_id(group_id, enterprise_id)
        except ApiException as e:
            self.app.log.debug(f"Failed to show details of an group: {e}")
            self.app.log.error(f"Failed to show details of an group, reason: {e.reason}")
            return

        if not self.app.pargs.json:
            renderable = self._group_basic_response(response)
            print(white(f"\tGROUP DETAILS", bold=True))
            self.app.render(renderable, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="fancy_grid")
        else:
            renderable = self._group_basic_response(response, OutputFormat.JSON)
            print(white(f"GROUP DETAILS", bold=True))
            self.app.render(renderable, format=OutputFormat.JSON.value)

    @ex(
        help='Show or unset the active group',
        arguments=[
            (['-u', '--unset'],
             {'help': 'Unset the active group',
              'action': 'store_true',
              'dest': 'unset'}),
            (['-j', '--json'],
             {'help': 'Render result in Json format',
              'action': 'store_true',
              'dest': 'json'})
        ]
    )
    def active(self):
        validate_creds_exists(self.app)
        db = DBWrapper(self.app.creds)
        group_client = APIClient(db.get_configure()).get_group_api_client()
        enterprise_id = db.get_enterprise_id()

        group_id = None
        group = db.get_group()
        if group:
            group_id = group.get('id', None)

        if self.app.pargs.unset:
            if not group_id:
                self.app.log.info('Not set the active group.')
                return

            db.unset_group()
            self.app.log.info(f'Unset the active group {group_id}')
            return

        if not group_id:
            self.app.log.info("Not set the active group.")
            return

        try:
            response = group_client.get_group_by_id(group_id, enterprise_id)
        except ApiException as e:
            self.app.log.debug(f"Failed to show or unset the active group: {e}")
            self.app.log.error(f"Failed to show or unset the active group, reason: {e.reason}")
            return

        if not self.app.pargs.json:
            print(white(f"\tGROUP DETAILS of {response.name}", bold=True))
            renderable = self._group_basic_response(response)
            self.app.render(renderable, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="fancy_grid")
        else:
            print(white(f"GROUP DETAILS of {response.name}", bold=True))
            renderable = self._group_basic_response(response, OutputFormat.JSON)
            self.app.render(renderable, format=OutputFormat.JSON.value)

    @ex(
        help='Create group',
        arguments=[
            (['-n', '--name'],
             {'help': 'Group name',
              'action': 'store',
              'dest': 'name'}),
            (['-j', '--json'],
             {'help': 'Render result in Json format',
              'action': 'store_true',
              'dest': 'json'})
        ]
    )
    def create(self):
        validate_creds_exists(self.app)
        db = DBWrapper(self.app.creds)
        group_client = APIClient(db.get_configure()).get_group_api_client()
        enterprise_id = db.get_enterprise_id()

        if self.app.pargs.name:
            data = DeviceGroup(name=self.app.pargs.name)
        else:
            self.app.log.error('name cannot be empty.')
            return

        try:
            response = group_client.create_group(enterprise_id, data)
        except ApiException as e:
            self.app.log.debug(f"Failed to create a group: {e}")
            self.app.log.error(f"Failed to create a group, reason: {e.reason}")
            return

        if not self.app.pargs.json:
            renderable = self._group_basic_response(response)
            print(white(f"\tGROUP DETAILS", bold=True))
            self.app.render(renderable, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="fancy_grid")
        else:
            renderable = self._group_basic_response(response, OutputFormat.JSON)
            print(white(f"GROUP DETAILS", bold=True))
            self.app.render(renderable, format=OutputFormat.JSON.value)

    @ex(
        help='Update group details',
        arguments=[
            (['group_id'],
             {'help': 'Group id',
              'action': 'store'}),
            (['-n', '--name'],
             {'help': 'Group name',
              'action': 'store',
              'dest': 'name'}),
            (['-j', '--json'],
             {'help': 'Render result in Json format',
              'action': 'store_true',
              'dest': 'json'})
        ]
    )
    def update(self):
        validate_creds_exists(self.app)
        db = DBWrapper(self.app.creds)
        group_client = APIClient(db.get_configure()).get_group_api_client()
        enterprise_id = db.get_enterprise_id()
        data = DeviceGroupUpdate()

        group_id = self.app.pargs.group_id

        if self.app.pargs.name:
            data.name = self.app.pargs.name
        else:
            self.app.log.error('name cannot be empty.')
            return

        try:
            response = group_client.partial_update_group(group_id, enterprise_id, data)
        except ApiException as e:
            self.app.log.debug(f"Failed to update details of a group: {e}")
            self.app.log.error(f"Failed to update details of a group, reason: {e.reason}")
            return

        if not self.app.pargs.json:
            renderable = self._group_basic_response(response)
            print(white(f"\tGROUP DETAILS", bold=True))
            self.app.render(renderable, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="fancy_grid")
        else:
            renderable = self._group_basic_response(response, OutputFormat.JSON)
            print(white(f"GROUP DETAILS", bold=True))
            self.app.render(renderable, format=OutputFormat.JSON.value)

    @ex(
        help='Delete group',
        arguments=[
            (['group_id'],
             {'help': 'Group id',
              'action': 'store'}),
        ]
    )
    def delete(self):
        group_id = self.app.pargs.group_id

        validate_creds_exists(self.app)
        db = DBWrapper(self.app.creds)
        group_client = APIClient(db.get_configure()).get_group_api_client()
        enterprise_id = db.get_enterprise_id()

        try:
            group_client.delete_group(group_id, enterprise_id)
            self.app.log.info(f"Group with id : {group_id} deleted successfully")

            # Unset current group if matching
            group = db.get_group()
            if group and group.get('id') and group_id == group.get('id'):
                db.unset_group()
                self.app.log.debug(f'Unset the current group {group_id}')
        except ApiException as e:
            self.app.log.debug(f"Failed to delete group: {e}")
            self.app.log.error(f"Failed to delete group, reason: {e.reason}")
            return

    def _get_group_device_ids(self, device_client, enterprise_id, group_id):
        device_ids = []
        try:
            counter = 0
            limit = 100
            while True:
                offset = limit * counter
                response = device_client.get_all_devices(enterprise_id, group=group_id, limit=limit, offset=offset)
                if not response.results or len(response.results) == 0:
                    break

                device_ids.extend([device.id for device in response.results])
                counter += 1
        except ApiException as e:
            self.app.log.debug(f"Failed to list device by group: {e}")
            self.app.log.error(f"Failed to get devices in the group, reason: {e.reason}")
            return None

        return device_ids

    @ex(
        help='Add devices to group',
        arguments=[
            (['-g', '--group'],
             {'help': 'Group id',
              'action': 'store',
              'dest': 'group'}),
            (['-d', '--devices'],
             {'help': 'List of devices, space separated',
              'nargs': "*",
              'type': str,
              'dest': 'devices'}),
            (['-j', '--json'],
             {'help': 'Render result in Json format',
              'action': 'store_true',
              'dest': 'json'})
        ]
    )
    def add(self):
        validate_creds_exists(self.app)
        db = DBWrapper(self.app.creds)
        group_client = APIClient(db.get_configure()).get_group_api_client()
        enterprise_id = db.get_enterprise_id()
        data = DeviceGroupUpdate()

        if self.app.pargs.group:
            group_id = self.app.pargs.group
        else:
            group = db.get_group()
            if not group or not group.get('id'):
                self.app.log.info('Not set the active group.')
                return

            group_id = group.get('id')

        devices = self.app.pargs.devices
        if not devices or len(devices) == 0:
            self.app.log.error('devices cannot be empty.')
            return

        device_client = APIClient(db.get_configure()).get_device_api_client()
        device_ids = self._get_group_device_ids(device_client, enterprise_id, group_id)
        if device_ids is None:
            return

        device_ids.extend(devices)
        data.device_ids = device_ids
        try:
            response = group_client.partial_update_group(group_id, enterprise_id, data)
        except ApiException as e:
            self.app.log.debug(f"Failed to add device into a group: {e}")
            self.app.log.error(f"Failed to add device into a group, reason: {e.reason}")
            return

        if not self.app.pargs.json:
            renderable = self._group_basic_response(response)
            print(white(f"\tGROUP DETAILS", bold=True))
            self.app.render(renderable, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="fancy_grid")
        else:
            renderable = self._group_basic_response(response, OutputFormat.JSON)
            print(white(f"GROUP DETAILS", bold=True))
            self.app.render(renderable, format=OutputFormat.JSON.value)

    @ex(
        help='Remove devices from group',
        arguments=[
            (['-g', '--group'],
             {'help': 'Group id',
              'action': 'store',
              'dest': 'group'}),
            (['-d', '--devices'],
             {'help': 'List of devices, space separated',
              'nargs': "*",
              'type': str,
              'dest': 'devices'}),
            (['-j', '--json'],
             {'help': 'Render result in Json format',
              'action': 'store_true',
              'dest': 'json'})
        ]
    )
    def remove(self):
        validate_creds_exists(self.app)
        db = DBWrapper(self.app.creds)
        group_client = APIClient(db.get_configure()).get_group_api_client()
        enterprise_id = db.get_enterprise_id()
        data = DeviceGroupUpdate()

        if self.app.pargs.group:
            group_id = self.app.pargs.group
        else:
            group = db.get_group()
            if not group or not group.get('id'):
                self.app.log.info('Not set the active group.')
                return

            group_id = group.get('id')

        devices = self.app.pargs.devices
        if not devices or len(devices) == 0:
            self.app.log.error('devices cannot be empty.')
            return

        device_client = APIClient(db.get_configure()).get_device_api_client()
        current_device_ids = self._get_group_device_ids(device_client, enterprise_id, group_id)
        if current_device_ids is None:
            return

        latest_devices = list(set(current_device_ids) - set(devices))
        data.device_ids = latest_devices
        try:
            response = group_client.partial_update_group(group_id, enterprise_id, data)
        except ApiException as e:
            self.app.log.debug(f"Failed to remove device from group: {e}")
            self.app.log.error(f"Failed to remove device from group, reason: {e.reason}")
            return

        if not self.app.pargs.json:
            renderable = self._group_basic_response(response)
            print(white(f"\tGROUP DETAILS", bold=True))
            self.app.render(renderable, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="fancy_grid")
        else:
            renderable = self._group_basic_response(response, OutputFormat.JSON)
            print(white(f"GROUP DETAILS", bold=True))
            self.app.render(renderable, format=OutputFormat.JSON.value)
