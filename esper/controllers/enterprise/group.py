from cement import Controller, ex
from esperclient import DeviceGroup, DeviceGroupUpdate, DeviceGroupPartialUpdate
from esperclient.rest import ApiException

from esper.controllers.enums import OutputFormat, DeviceState
from esper.ext.api_client import APIClient
from esper.ext.db_wrapper import DBWrapper
from esper.ext.utils import validate_creds_exists, parse_error_message


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
            self.app.log.error(f"[group-list] Failed to list groups: {e}")
            self.app.render(f"ERROR: {parse_error_message(self.app, e)}")
            return

        if not self.app.pargs.json:
            groups = []

            label = {
                'id': "ID",
                'name': "NAME",
                'parent_id': "PARENT ID",
                'device_count': "DEVICE COUNT",      
            }

            for group in response.results:
                if group.parent:
                    parent = group.parent.split('/')[-2]
                else:
                    parent = group.parent
                groups.append(
                    {
                        label['id']: group.id,
                        label['name']: group.name,
                        label['device_count']: group.device_count if group.device_count else 0,
                        label['parent_id']: parent,
                    }
                )
            self.app.render(f"Number of Groups: {response.count}")
            self.app.render(groups, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")
        else:
            groups = []
            for group in response.results:
                if group.parent:
                    parent = group.parent.split('/')[-2]
                else:
                    parent = group.parent
                groups.append(
                    {
                        'id': group.id,
                        'name': group.name,
                        'device_count': group.device_count if group.device_count else 0,
                        'parent_id': parent,
                    }
                )
            self.app.render(f"Number of Groups: {response.count}")
            self.app.render(groups, format=OutputFormat.JSON.value)

    def _group_basic_response(self, group, format=OutputFormat.TABULATED):
        if group.parent:
            parent = group.parent.split('/')[-2]
        else:
            parent = group.parent

        if format == OutputFormat.TABULATED:
            title = "TITLE"
            details = "DETAILS"
            renderable = [
                {title: 'id', details: group.id},
                {title: 'name', details: group.name},
                {title: 'parent_id', details: parent},
                {title: 'device_count', details: group.device_count},
                {title: 'path', details: group.path},
                {title: 'children_count', details: group.children_count},      
            ]
        else:
            renderable = {
                'id': group.id,
                'name': group.name,
                'parent': parent,
                'device_count': group.device_count,
                'path': group.path,
                'children_count': group.children_count,
            }

        return renderable

    @ex(
        help='Show group details and set as active group',
        arguments=[
            (['group_name'],
             {'help': 'Group name',
              'action': 'store'}),
            (['-id', '--groupid'],
             {'help': 'Group id',
              'action': 'store',
              'dest': 'group_id'}),
            (['-a', '--active'],
             {'help': 'Set this group as active for further group specific commands',
              'action': 'store_true',
              'dest': 'active'}),
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

        if self.app.pargs.group_id:
            group_id = self.app.pargs.group_id
            try:
                response = group_client.get_group_by_id(group_id, enterprise_id)
                group_name = self.app.pargs.group_name
                if group_name != response.name:
                    self.app.log.debug(f"[group-show] Group does not exist with id: {group_id} and name: {group_name}")
                    self.app.render(f"Group does not exist with id {group_id} and name {group_name} \n")
                    return
            except ApiException as e:
                self.app.log.error(f"[group-show] Group does not exist with id {group_id}: {e}")
                self.app.render(f"ERROR: {parse_error_message(self.app, e)} \n")
                return

        else:
            group_name = self.app.pargs.group_name
            kwargs = {'name': group_name}
            try:
                search_response = group_client.get_all_groups(enterprise_id, limit=1, offset=0, **kwargs)
                response = None
                for group in search_response.results:
                    if group.name == group_name:
                        response = group
                        break

                if not response:
                    self.app.log.info(f'[group-show] Group does not exist with name {group_name}')
                    self.app.render(f'Group does not exist with name {group_name} \n')
                    return
            except ApiException as e:
                self.app.log.error(f"[group-show] Failed to list groups: {e}")
                self.app.render(f"ERROR: {parse_error_message(self.app, e)} \n")
                return

        if self.app.pargs.active:
            db.set_group({'id': response.id, 'name': response.name})

        if not self.app.pargs.json:
            renderable = self._group_basic_response(response)
            self.app.render(renderable, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")
        else:
            renderable = self._group_basic_response(response, OutputFormat.JSON)
            self.app.render(renderable, format=OutputFormat.JSON.value)

    @ex(
        help='Set or show the active group',
        arguments=[
            (['-n', '--name'],
             {'help': 'Group name.',
              'action': 'store',
              'dest': 'name'}),
            (['-id', '--groupid'],
             {'help': 'Group id.',
              'action': 'store',
              'dest': 'group_id'}),
            (['-j', '--json'],
             {'help': 'Render result in Json format',
              'action': 'store_true',
              'dest': 'json'})
        ]
    )
    def set_active(self):
        validate_creds_exists(self.app)
        db = DBWrapper(self.app.creds)
        group_client = APIClient(db.get_configure()).get_group_api_client()
        enterprise_id = db.get_enterprise_id()

        if self.app.pargs.group_id:
            group_id = self.app.pargs.group_id
            try:
                response = group_client.get_group_by_id(group_id, enterprise_id)
                group_name = self.app.pargs.name
                if group_name != response.name:
                    self.app.log.debug(f"[group-active] Group does not exist with id: {group_id} and name: {group_name}")
                    self.app.render(f"Group does not exist with id {group_id} and name {group_name} \n")
                    return
                db.set_group({'id': response.id, 'name': response.name})
            except ApiException as e:
                self.app.log.error(f"[group-active] Group doesn't exist with id {group_id}: {e}")
                self.app.render(f"ERROR: {parse_error_message(self.app, e)} \n")
                return

        elif self.app.pargs.name:
            group_name = self.app.pargs.name
            kwargs = {'name': group_name}
            try:
                search_response = group_client.get_all_groups(enterprise_id, limit=1, offset=0, **kwargs)
                if not search_response.results or len(search_response.results) == 0:
                    self.app.log.info(f'[group-active] Group does not exist with name {group_name}')
                    self.app.render(f'Group does not exist with name {group_name} \n')
                    return
                response = search_response.results[0]
                db.set_group({'id': response.id, 'name': group_name})
            except ApiException as e:
                self.app.log.error(f"[group-active] Failed to list groups: {e}")
                self.app.render(f"ERROR: {parse_error_message(self.app, e)} \n")
                return
        else:
            group = db.get_group()
            if group is None or group.get('name') is None:
                self.app.log.debug('[group-active] There is no active group.')
                self.app.render('There is no active group.')
                return

            group_id = group.get('id')
            try:
                response = group_client.get_group_by_id(group_id, enterprise_id)
            except ApiException as e:
                self.app.log.error(f"[group-active] Failed to show active group: {e}")
                self.app.render(f"ERROR: {parse_error_message(self.app, e)} \n")
                return

        if not self.app.pargs.json:
            renderable = self._group_basic_response(response)
            self.app.render(renderable, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")
        else:
            renderable = self._group_basic_response(response, OutputFormat.JSON)
            self.app.render(renderable, format=OutputFormat.JSON.value)

    @ex(
        help='Unset the current active group',
        arguments=[]
    )
    def unset_active(self):
        validate_creds_exists(self.app)
        db = DBWrapper(self.app.creds)

        group = db.get_group()
        if group is None or group.get('name') is None:
            self.app.log.debug('[group-active] There is no active group.')
            self.app.render('There is no active group.')
            return

        db.unset_group()
        self.app.log.debug(f"[group-active] Unset the active group with id: {group.get('id')} and name: {group.get('name')}")
        self.app.render(f"Unset the active group with id: {group.get('id')} and name: {group.get('name')}\n")

    @ex(
        help='Create group',
        arguments=[
            (['-n', '--name'],
             {'help': 'Group name',
              'action': 'store',
              'dest': 'name'}),
            (['-pid', '--parentid'],
             {'help': 'Parent Id',
              'action': 'store',
              'dest': 'parent'}),
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
            if self.app.pargs.parent:
                data = DeviceGroup(name=self.app.pargs.name, parent=self.app.pargs.parent)
            else:
                data = DeviceGroup(name=self.app.pargs.name)
        else:
            self.app.log.debug('[group-create] name cannot be empty.')
            self.app.render('name cannot be empty. \n')
            return

        try:
            response = group_client.create_group(enterprise_id, data)
        except ApiException as e:
            self.app.log.error(f"[group-create] Failed to create a group: {e}")
            self.app.render(f"ERROR: {parse_error_message(self.app, e)} \n")
            return

        if not self.app.pargs.json:
            renderable = self._group_basic_response(response)
            self.app.render(renderable, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")
        else:
            renderable = self._group_basic_response(response, OutputFormat.JSON)
            self.app.render(renderable, format=OutputFormat.JSON.value)

    @ex(
        help='Update group details',
        arguments=[
            (['name'],
             {'help': 'Group current name',
              'action': 'store'}),
            (['-id', '--groupid'],
             {'help': 'Group id',
              'action': 'store',
              'dest': 'group_id'}),
            (['-n', '--name'],
             {'help': 'Group new name',
              'action': 'store',
              'dest': 'new_name'}),
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
        data = DeviceGroupPartialUpdate()

        if self.app.pargs.group_id:
            group_id = self.app.pargs.group_id
            try:
                response = group_client.get_group_by_id(group_id, enterprise_id)
                group_name = self.app.pargs.name
                if group_name != response.name:
                    self.app.log.debug(f"[group-show] Group does not exist with id: {group_id} and name: {group_name}")
                    self.app.render(f"Group does not exist with id {group_id} and name {group_name} \n")
                    return
            except ApiException as e:
                self.app.log.error(f"[group-update] Group doesn't exist with id {group_id}: {e}")
                self.app.render(f"ERROR: {parse_error_message(self.app, e)} \n")
                return

        elif self.app.pargs.name:
            group_name = self.app.pargs.name
            kwargs = {'name': group_name}
            try:
                search_response = group_client.get_all_groups(enterprise_id, limit=1, offset=0, **kwargs)
                if not search_response.results or len(search_response.results) == 0:
                    self.app.log.debug(f'[group-update] Group does not exist with name {group_name}')
                    self.app.render(f'Group does not exist with name {group_name} \n')
                    return
                response = search_response.results[0]
                group_id = response.id
            except ApiException as e:
                self.app.log.error(f"[group-update] Failed to list groups: {e}")
                self.app.render(f"ERROR: {parse_error_message(self.app, e)} \n")
                return

        if self.app.pargs.new_name:
            data.name = self.app.pargs.new_name
        else:
            self.app.log.debug('[group-update] new name cannot be empty.')
            self.app.render('new name cannot be empty. \n')
            return

        try:
            response = group_client.partial_update_group(group_id, enterprise_id, data)
        except ApiException as e:
            self.app.log.error(f"[group-update] Failed to update details of a group: {e}")
            self.app.render(f"ERROR: {parse_error_message(self.app, e)} \n")
            return

        if not self.app.pargs.json:
            renderable = self._group_basic_response(response)
            self.app.render(renderable, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")
        else:
            renderable = self._group_basic_response(response, OutputFormat.JSON)
            self.app.render(renderable, format=OutputFormat.JSON.value)

    @ex(
        help='Delete group',
        arguments=[
            (['name'],
             {'help': 'Group name',
              'action': 'store'}),
            (['-id', '--groupid'],
             {'help': 'Group id',
              'action': 'store',
              'dest': 'group_id'}),
        ]
    )
    def delete(self):
        validate_creds_exists(self.app)
        db = DBWrapper(self.app.creds)
        group_client = APIClient(db.get_configure()).get_group_api_client()
        enterprise_id = db.get_enterprise_id()

        if self.app.pargs.group_id:
            group_id = self.app.pargs.group_id
            try:
                response = group_client.get_group_by_id(group_id, enterprise_id)
                group_name = self.app.pargs.name
                if group_name != response.name:
                    self.app.log.debug(f"[group-show] Group does not exist with id: {group_id} and name: {group_name}")
                    self.app.render(f"Group does not exist with id {group_id} and name {group_name} \n")
                    return
            except ApiException as e:
                self.app.log.error(f"[group-delete] Group doesn't exist with id {group_id}: {e}")
                self.app.render(f"ERROR: {parse_error_message(self.app, e)} \n")
                return

        elif self.app.pargs.name:
            group_name = self.app.pargs.name
            kwargs = {'name': group_name}
            try:
                search_response = group_client.get_all_groups(enterprise_id, limit=1, offset=0, **kwargs)
                if not search_response.results or len(search_response.results) == 0:
                    self.app.log.debug(f'[group-delete] Group does not exist with name {group_name}')
                    self.app.render(f'Group does not exist with name {group_name} \n')
                    return
                response = search_response.results[0]
                group_id = response.id
            except ApiException as e:
                self.app.log.error(f"[group-update] Failed to list groups: {e}")
                self.app.render(f"ERROR: {parse_error_message(self.app, e)} \n")
                return

        try:
            group_client.delete_group(group_id, enterprise_id)
            self.app.log.debug(f"[group-update] Group with name {group_name} deleted successfully")
            self.app.render(f"Group with name {group_name} deleted successfully \n")

            # Unset current group if matching
            group = db.get_group()
            if group and group.get('id') and group_id == group.get('id'):
                db.unset_group()
                self.app.log.debug(f'[group-update] Unset the active group {group_name}')
        except ApiException as e:
            self.app.log.error(f"[group-update] Failed to delete group: {e}")
            self.app.render(f"ERROR: {parse_error_message(self.app, e)} \n")
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
            self.app.log.error(f"[_get_group_device_ids] Failed to list device by group: {e}")
            self.app.render(f"ERROR: {parse_error_message(self.app, e)} \n")
            return None

        return device_ids

    @ex(
        help='Add devices to group',
        arguments=[
            (['-g', '--group'],
             {'help': 'Group name',
              'action': 'store',
              'dest': 'group'}),
            (['-id', '--groupid'],
             {'help': 'Group id',
              'action': 'store',
              'dest': 'group_id'}),
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
        data = DeviceGroupPartialUpdate()
        action = 'add'

        if self.app.pargs.group_id:
            group_id = self.app.pargs.group_id
            try:
                response = group_client.get_group_by_id(group_id, enterprise_id)
                group_name = self.app.pargs.group
                if group_name != response.name:
                    self.app.log.debug(f"[group-add] Group does not exist with id: {group_id} and name: {group_name}")
                    self.app.render(f"Group does not exist with id {group_id} and name {group_name} \n")
                    return
            except ApiException as e:
                self.app.log.error(f"[group-add] Group doesn't exist with id {group_id}: {e}")
                self.app.render(f"ERROR: {parse_error_message(self.app, e)} \n")
                return

        elif self.app.pargs.group:
            group_name = self.app.pargs.group
            kwargs = {'name': group_name}
            try:
                search_response = group_client.get_all_groups(enterprise_id, limit=1, offset=0, **kwargs)
                if not search_response.results or len(search_response.results) == 0:
                    self.app.log.debug(f'[group-add] Group does not exist with name {group_name}')
                    self.app.render(f'Group does not exist with name {group_name} \n')
                    return
                response = search_response.results[0]
                group_id = response.id
            except ApiException as e:
                self.app.log.error(f"[group-add] Failed to list groups: {e}")
                self.app.render(f"ERROR: {parse_error_message(self.app, e)} \n")
                return

        else:
            group = db.get_group()
            if group is None or group.get('name') is None:
                self.app.log.debug('[group-add] There is no active group.')
                self.app.render('There is no active group. \n')
                return

            group_id = group.get('id')

        devices = self.app.pargs.devices
        if not devices or len(devices) == 0:
            self.app.log.debug('[group-add] devices cannot be empty.')
            self.app.render('devices cannot be empty. \n')
            return

        if len(devices) > 1000:
            self.app.log.debug('[group-add] Cannot add more than 1000 devices at a time.')
            self.app.render('Cannot add more than 1000 devices at a time. \n')
            return

        device_client = APIClient(db.get_configure()).get_device_api_client()
        request_device_ids = []
        for device_name in devices:
            kwargs = {'name': device_name}
            try:
                search_response = device_client.get_all_devices(enterprise_id, limit=1, offset=0, **kwargs)
                if not search_response.results or len(search_response.results) == 0:
                    self.app.debug(f'[group-add] Device does not exist with name {device_name}')
                    return
                response = search_response.results[0]
                request_device_ids.append(response.id)
            except ApiException as e:
                self.app.log.error(f"[group-add] Failed to list devices: {e}")
                self.app.render(f"ERROR: {parse_error_message(self.app, e)} \n")
                return

        device_client = APIClient(db.get_configure()).get_device_api_client()
        device_ids = self._get_group_device_ids(device_client, enterprise_id, group_id)
        if device_ids is None:
            return

        device_ids.extend(request_device_ids)
        data.device_ids = device_ids
        try:
            response = group_client.partial_update_group(group_id, enterprise_id, data, action=action)
        except ApiException as e:
            self.app.log.error(f"[group-add]  Failed to add device into a group: {e}")
            self.app.render(f"ERROR: {parse_error_message(self.app, e)} \n")
            return

        if not self.app.pargs.json:
            renderable = self._group_basic_response(response)
            self.app.render(renderable, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")
        else:
            renderable = self._group_basic_response(response, OutputFormat.JSON)
            self.app.render(renderable, format=OutputFormat.JSON.value)

    @ex(
        help='Remove devices from group',
        arguments=[
            (['-g', '--group'],
             {'help': 'Group name',
              'action': 'store',
              'dest': 'group'}),
            (['-id', '--groupid'],
             {'help': 'Group id',
              'action': 'store',
              'dest': 'group_id'}),
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
        data = DeviceGroupPartialUpdate()
        action = 'remove'

        if self.app.pargs.group_id:
            group_id = self.app.pargs.group_id
            try:
                response = group_client.get_group_by_id(group_id, enterprise_id)
                group_name = self.app.pargs.group
                if group_name != response.name:
                    self.app.log.debug(f"[group-remove] Group does not exist with id: {group_id} and name: {group_name}")
                    self.app.render(f"Group does not exist with id {group_id} and name {group_name} \n")
                    return
            except ApiException as e:
                self.app.log.error(f"[group-remove] Group doesn't exist with id {group_id}: {e}")
                self.app.render(f"ERROR: {parse_error_message(self.app, e)} \n")
                return

        elif self.app.pargs.group:
            group_name = self.app.pargs.group
            kwargs = {'name': group_name}
            try:
                search_response = group_client.get_all_groups(enterprise_id, limit=1, offset=0, **kwargs)
                if not search_response.results or len(search_response.results) == 0:
                    self.app.log.debug(f'[group-remove] Group does not exist with name {group_name}')
                    self.app.render(f'Group does not exist with name {group_name} \n')
                    return
                response = search_response.results[0]
                group_id = response.id
            except ApiException as e:
                self.app.log.error(f"[group-remove] Failed to list groups: {e}")
                self.app.render(f"ERROR: {parse_error_message(self.app, e)} \n")
                return

        else:
            group = db.get_group()
            if group is None or group.get('name') is None:
                self.app.log.debug('[group-remove] There is no active group.')
                self.app.render('There is no active group. \n')
                return

            group_id = group.get('id')

        devices = self.app.pargs.devices
        if not devices or len(devices) == 0:
            self.app.log.debug('[group-remove] devices cannot be empty.')
            self.app.render('devices cannot be empty. \n')
            return
        
        if len(devices) > 1000:
            self.app.log.debug('[group-remove] Cannot remove more than 1000 devices at a time.')
            self.app.render('Cannot remove more than 1000 devices at a time. \n')
            return

        device_client = APIClient(db.get_configure()).get_device_api_client()
        request_device_ids = []
        for device_name in devices:
            kwargs = {'name': device_name}
            try:
                search_response = device_client.get_all_devices(enterprise_id, limit=1, offset=0, **kwargs)
                if not search_response.results or len(search_response.results) == 0:
                    self.app.log.debug(f'[group-remove] Device does not exist with name {device_name}')
                    self.app.render(f'Device does not exist with name {device_name} \n')
                    return
                response = search_response.results[0]
                request_device_ids.append(response.id)
            except ApiException as e:
                self.app.log.error(f"Failed to list devices: {e}")
                self.app.render(f"ERROR: {parse_error_message(self.app, e)} \n")
                return

        device_client = APIClient(db.get_configure()).get_device_api_client()
        current_device_ids = self._get_group_device_ids(device_client, enterprise_id, group_id)

        extra_devices = list(set(request_device_ids) - set(current_device_ids))

        if extra_devices:
            self.app.log.debug('[group-remove] The given devices are not present in the group.')
            self.app.render('The given devices are not present in the group. \n')
            return

        if current_device_ids is None:
            return

        data.device_ids = request_device_ids
        try:
            response = group_client.partial_update_group(group_id, enterprise_id, data, action=action)
        except ApiException as e:
            self.app.log.error(f"[group-remove] Failed to remove device from group: {e}")
            self.app.render(f"ERROR: {parse_error_message(self.app, e)} \n")
            return

        if not self.app.pargs.json:
            renderable = self._group_basic_response(response)
            self.app.render(renderable, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")
        else:
            renderable = self._group_basic_response(response, OutputFormat.JSON)
            self.app.render(renderable, format=OutputFormat.JSON.value)

    @ex(
        help='List group devices',
        arguments=[
            (['-g', '--group'],
             {'help': 'Group name',
              'action': 'store',
              'dest': 'group'}),
            (['-id', '--groupid'],
             {'help': 'Group id',
              'action': 'store',
              'dest': 'group_id'}),
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
    def devices(self):
        validate_creds_exists(self.app)
        db = DBWrapper(self.app.creds)
        device_client = APIClient(db.get_configure()).get_device_api_client()
        group_client = APIClient(db.get_configure()).get_group_api_client()
        enterprise_id = db.get_enterprise_id()

        if self.app.pargs.group_id:
            group_id = self.app.pargs.group_id
            try:
                response = group_client.get_group_by_id(group_id, enterprise_id)
                group_name = self.app.pargs.group
                if group_name != response.name:
                    self.app.log.debug(f"[group-devices] Group does not exist with id: {group_id} and name: {group_name}")
                    self.app.render(f"Group does not exist with id {group_id} and name {group_name} \n")
                    return
            except ApiException as e:
                self.app.log.error(f"[group-devices] Group doesn't exist with id {group_id}: {e}")
                self.app.render(f"ERROR: {parse_error_message(self.app, e)} \n")
                return

        elif self.app.pargs.group:
            group_name = self.app.pargs.group
            kwargs = {'name': group_name}
            try:
                search_response = group_client.get_all_groups(enterprise_id, limit=1, offset=0, **kwargs)
                if not search_response.results or len(search_response.results) == 0:
                    self.app.log.debug(f'[group-devices] Group does not exist with name {group_name}')
                    self.app.render(f'Group does not exist with name {group_name} \n')
                    return
                response = search_response.results[0]
                group_id = response.id
            except ApiException as e:
                self.app.log.error(f"[group-devices] Failed to list groups: {e}")
                self.app.render(f"ERROR: {parse_error_message(self.app, e)} \n")
                return

        else:
            group = db.get_group()
            if group is None or group.get('name') is None:
                self.app.log.debug('[group-devices] There is no active group.')
                self.app.render('There is no active group. \n')
                return

            group_id = group.get('id')

        limit = self.app.pargs.limit
        offset = self.app.pargs.offset

        try:
            response = device_client.get_all_devices(enterprise_id, group=group_id, limit=limit, offset=offset)
        except ApiException as e:
            self.app.log.error(f"[group-devices] Failed to list group devices: {e}")
            self.app.render(f"ERROR: {parse_error_message(self.app, e)} \n")
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
                name = device.device_name
                if device.alias_name and device.alias_name != '':
                    name = device.alias_name
                tags = ''
                if device.tags and len(device.tags) > 0:
                    tags = ', '.join(device.tags)
                devices.append(
                    {
                        label['id']: device.id,
                        label['name']: name,
                        label['model']: device.hardware_info.get("manufacturer"),
                        label['state']: DeviceState(device.status).name,
                        label['tags']: tags
                    }
                )
            self.app.render(devices, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")
        else:
            devices = []
            for device in response.results:
                name = device.device_name
                if device.alias_name and device.alias_name != '':
                    name = device.alias_name
                devices.append(
                    {
                        'id': device.id,
                        'device': name,
                        'model': device.hardware_info.get("manufacturer"),
                        'state': DeviceState(device.status).name,
                        'tags': device.tags
                    }
                )
            self.app.render(devices, format=OutputFormat.JSON.value)


    @ex(
        help='Move a group',
        arguments=[
            (['-g', '--group'],
             {'help': 'Group name',
              'action': 'store',
              'dest': 'group'}),
            (['-id', '--groupid'],
             {'help': 'Group id',
              'action': 'store',
              'dest': 'group_id'}),
            (['-p', '--parent'],
             {'help': 'Parent name',
              'action': 'store',
              'dest': 'parent'}),
            (['-pid', '--parentid'],
             {'help': 'Parent Id',
              'action': 'store',
              'dest': 'parent_id'}),
            (['-j', '--json'],
             {'help': 'Render result in Json format',
              'action': 'store_true',
              'dest': 'json'})
        ]
    )
    def move(self):
        validate_creds_exists(self.app)
        db = DBWrapper(self.app.creds)
        group_client = APIClient(db.get_configure()).get_group_api_client()
        enterprise_id = db.get_enterprise_id()
        action = 'move'

        if self.app.pargs.group_id:
            group_id = self.app.pargs.group_id
            try: 
                response = group_client.get_group_by_id(group_id, enterprise_id)
                group_name = self.app.pargs.group
                if group_name != response.name:
                    self.app.log.debug(f"[group-move] Group does not exist with id: {group_id} and name: {group_name}")
                    self.app.render(f"Group does not exist with id {group_id} and name {group_name} \n")
                    return
            except ApiException as e:
                self.app.log.error(f"[group-move] Group does not exist with id {group_id}: {e}")
                self.app.render(f"ERROR: {parse_error_message(self.app, e)} \n")
                return

        elif self.app.pargs.group:
            group_name = self.app.pargs.group
            kwargs = {'name': group_name}
            try:
                search_response = group_client.get_all_groups(enterprise_id, limit=1, offset=0, **kwargs)
                if not search_response.results or len(search_response.results) == 0:
                    self.app.log.debug(f'[group-move] Group does not exist with name {group_name}')
                    self.app.render(f'Group does not exist with name {group_name} \n')
                    return
                response = search_response.results[0]
                group_id = response.id
            except ApiException as e:
                self.app.log.error(f"[group-move] Failed to list groups: {e}")
                self.app.render(f"ERROR: {parse_error_message(self.app, e)} \n")
                return

        else:
            group = db.get_group()
            if group is None or group.get('name') is None:
                self.app.log.debug('[group-move] There is no active group.')
                self.app.render('There is no active group. \n')
                return

            group_id = group.get('id')
            group_name = group.get('name')
        
        if not (self.app.pargs.parent_id or self.app.pargs.parent):
            self.app.log.debug('[group-move] Either parent name or parent id should be present.')
            self.app.render('Either parent name or parent id should be present. \n')
            return

        if self.app.pargs.parent_id:
            parent_id = self.app.pargs.parent_id
            try:
                response = group_client.get_group_by_id(parent_id, enterprise_id)
                parent_name = self.app.pargs.parent
                if parent_name != response.name:
                    self.app.log.debug(f"[group-active] Group does not exist with id: {parent_id} and name: {parent_name}")
                    self.app.render(f"Group does not exist with id {parent_id} and name {parent_name} \n")
                    return
                data = DeviceGroupUpdate(name=group_name, parent=parent_id)
            except ApiException as e:
                self.app.log.error(f"[group-move] Group does not exist with id {parent_id}: {e}")
                self.app.render(f"ERROR: {parse_error_message(self.app, e)} \n")
                return
        
        elif self.app.pargs.parent:
            parent_name = self.app.pargs.parent
            kwargs = {'name': parent_name}
            try:
                search_response = group_client.get_all_groups(enterprise_id, limit=1, offset=0, **kwargs)
                if not search_response.results or len(search_response.results) == 0:
                    self.app.log.debug(f'[group-move] Group does not exist with name {parent_name}')
                    self.app.render(f'Group does not exist with name {parent_name} \n')
                    return
                response = search_response.results[0]
                parent_id = response.id
                data = DeviceGroupUpdate(name=group_name, parent=parent_id)
            except ApiException as e:
                self.app.log.error(f"[group-move] Failed to list groups: {e}")
                self.app.render(f"ERROR: {parse_error_message(self.app, e)} \n")
                return
        
        try:
            response = group_client.partial_update_group(group_id, enterprise_id, data, action=action)
        except ApiException as e:
            self.app.log.error(f"[group-move] Failed to move device from group: {e}")
            self.app.render(f"ERROR: {parse_error_message(self.app, e)} \n")
            return

        if not self.app.pargs.json:
            renderable = self._group_basic_response(response)
            self.app.render(renderable, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")
        else:
            renderable = self._group_basic_response(response, OutputFormat.JSON)
            self.app.render(renderable, format=OutputFormat.JSON.value)