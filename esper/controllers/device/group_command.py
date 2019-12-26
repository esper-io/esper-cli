from ast import literal_eval

from cement import ex, Controller
from esperclient import GroupCommandRequest
from esperclient.rest import ApiException

from esper.controllers.enums import OutputFormat, DeviceCommandEnum
from esper.ext.api_client import APIClient
from esper.ext.db_wrapper import DBWrapper
from esper.ext.utils import validate_creds_exists, parse_error_message


class GroupCommand(Controller):
    class Meta:
        label = 'group-command'

        # text displayed at the top of --help output
        description = 'Fire commands into a group of devices'

        # text displayed at the bottom of --help output
        epilog = 'Usage: espercli group-command'

        stacked_type = 'nested'
        stacked_on = 'base'

    def _command_basic_response(self, command, format=OutputFormat.TABULATED):
        valid_keys = ['id', 'command', 'state']

        success = None
        failed = None
        in_progress = None
        inactive = None

        success_list = []
        failed_list = []
        inactive_list = []
        in_progress_list = []
        if command.details:
            details = literal_eval(command.details)
            if details.get('success'):
                success_list.extend([device.get('name', None) for device in details.get('success')])
                if len(success_list) > 0:
                    success = '\n'.join(success_list)

            if details.get('failed'):
                failed_list.extend([device.get('name', None) for device in details.get('failed')])
                if len(failed_list) > 0:
                    failed = '\n'.join(failed_list)

            if details.get('inactive'):
                inactive_list.extend([device.get('name', None) for device in details.get('inactive')])
                if len(inactive_list) > 0:
                    inactive = '\n'.join(inactive_list)

            if details.get('acknowledge'):
                in_progress_list.extend([device.get('name', None) for device in details.get('acknowledge')])

            if details.get('initiate'):
                in_progress_list.extend([device.get('name', None) for device in details.get('initiate')])

            if details.get('in_progress'):
                in_progress_list.extend([device.get('name', None) for device in details.get('in_progress')])

            if details.get('timeout'):
                in_progress_list.extend([device.get('name', None) for device in details.get('timeout')])

            if len(in_progress_list) > 0:
                in_progress = '\n'.join(in_progress_list)

        if format == OutputFormat.TABULATED:
            title = "TITLE"
            details = "DETAILS"
            renderable = [{title: k, details: v} for k, v in command.to_dict().items() if k in valid_keys]
            renderable.extend([
                {title: 'success', details: success},
                {title: 'failed', details: failed},
                {title: 'in_progress', details: in_progress},
                {title: 'inactive', details: inactive}
            ])
        else:
            renderable = {k: v for k, v in command.to_dict().items() if k in valid_keys}
            renderable['success'] = success_list
            renderable['failed'] = failed_list
            renderable['in_progress'] = in_progress_list
            renderable['inactive'] = inactive_list

        return renderable

    @ex(
        help='Show command details',
        arguments=[
            (['command_id'],
             {'help': 'Group command id',
              'action': 'store'}),
            (['-g', '--group'],
             {'help': 'Group name',
              'action': 'store',
              'dest': 'group'}),
            (['-j', '--json'],
             {'help': 'Render result in Json format',
              'action': 'store_true',
              'dest': 'json'}),
        ]
    )
    def show(self):
        command_id = self.app.pargs.command_id
        validate_creds_exists(self.app)
        db = DBWrapper(self.app.creds)
        command_client = APIClient(db.get_configure()).get_group_command_api_client()
        enterprise_id = db.get_enterprise_id()
        group_client = APIClient(db.get_configure()).get_group_api_client()

        if self.app.pargs.group:
            group_name = self.app.pargs.group
            kwargs = {'name': group_name}
            try:
                search_response = group_client.get_all_groups(enterprise_id, limit=1, offset=0, **kwargs)
                if not search_response.results or len(search_response.results) == 0:
                    self.app.log.debug(f'[group-command-show] Group does not exist with name {group_name}')
                    self.app.render(f'Group does not exist with name {group_name}\n')
                    return
                response = search_response.results[0]
                group_id = response.id
            except ApiException as e:
                self.app.log.error(f"[group-command-show] ailed to list groups: {e}")
                self.app.render(f"ERROR: {parse_error_message(self.app, e)}\n")
                return

        else:
            group = db.get_group()
            if group is None or group.get('name') is None:
                self.app.log.debug('[group-command-show] There is no active group.')
                self.app.render('There is no active group.\n')
                return

            group_id = group.get('id')

        try:
            response = command_client.get_group_command(command_id, group_id, enterprise_id)
        except ApiException as e:
            self.app.log.error(f"[group-command-show] Failed to show details of group command: {e}")
            self.app.render(f"ERROR: {parse_error_message(self.app, e)}\n")
            return

        if not self.app.pargs.json:
            renderable = self._command_basic_response(response)
            self.app.render(renderable, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")
        else:
            renderable = self._command_basic_response(response, OutputFormat.JSON)
            self.app.render(renderable, format=OutputFormat.JSON.value)

    @ex(
        help='Install application version',
        arguments=[
            (['-g', '--group'],
             {'help': 'Group name',
              'action': 'store',
              'dest': 'group'}),
            (['-V', '--version'],
             {'help': 'Application version id',
              'action': 'store',
              'dest': 'version'}),
            (['-j', '--json'],
             {'help': 'Render result in Json format',
              'action': 'store_true',
              'dest': 'json'}),
        ]
    )
    def install(self):
        validate_creds_exists(self.app)
        db = DBWrapper(self.app.creds)
        command_client = APIClient(db.get_configure()).get_group_command_api_client()
        enterprise_id = db.get_enterprise_id()
        group_client = APIClient(db.get_configure()).get_group_api_client()

        if self.app.pargs.group:
            group_name = self.app.pargs.group
            kwargs = {'name': group_name}
            try:
                search_response = group_client.get_all_groups(enterprise_id, limit=1, offset=0, **kwargs)
                if not search_response.results or len(search_response.results) == 0:
                    self.app.log.debug(f'[group-command-install] Group does not exist with name {group_name}')
                    self.app.render(f'Group does not exist with name {group_name}\n')
                    return
                response = search_response.results[0]
                group_id = response.id
            except ApiException as e:
                self.app.log.error(f"[group-command-install] Failed to list groups: {e}")
                self.app.render(f"ERROR: {parse_error_message(self.app, e)}\n")
                return

        else:
            group = db.get_group()
            if group is None or group.get('name') is None:
                self.app.log.debug('[group-command-install] There is no active group.')
                self.app.render('There is no active group.\n')
                return

            group_id = group.get('id')

        version_id = self.app.pargs.version
        command_request = GroupCommandRequest(command_args={"app_version": version_id},
                                              command=DeviceCommandEnum.INSTALL.name)
        try:
            response = command_client.run_group_command(enterprise_id, group_id, command_request)
        except ApiException as e:
            self.app.log.error(f"[group-command-install] Failed to fire the install group command: {e}")
            self.app.render(f"ERROR: {parse_error_message(self.app, e)}\n")
            return

        if not self.app.pargs.json:
            renderable = self._command_basic_response(response)
            self.app.render(renderable, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")
        else:
            renderable = self._command_basic_response(response, OutputFormat.JSON)
            self.app.render(renderable, format=OutputFormat.JSON.value)

    @ex(
        help='Ping group of devices',
        arguments=[
            (['-g', '--group'],
             {'help': 'Group name',
              'action': 'store',
              'dest': 'group'}),
            (['-j', '--json'],
             {'help': 'Render result in Json format',
              'action': 'store_true',
              'dest': 'json'}),
        ]
    )
    def ping(self):
        validate_creds_exists(self.app)
        db = DBWrapper(self.app.creds)
        command_client = APIClient(db.get_configure()).get_group_command_api_client()
        enterprise_id = db.get_enterprise_id()
        group_client = APIClient(db.get_configure()).get_group_api_client()

        if self.app.pargs.group:
            group_name = self.app.pargs.group
            kwargs = {'name': group_name}
            try:
                search_response = group_client.get_all_groups(enterprise_id, limit=1, offset=0, **kwargs)
                if not search_response.results or len(search_response.results) == 0:
                    self.app.log.debug(f'[group-command-ping] Group does not exist with name {group_name}')
                    self.app.render(f'Group does not exist with name {group_name}\n')
                    return
                response = search_response.results[0]
                group_id = response.id
            except ApiException as e:
                self.app.log.error(f"[group-command-ping] Failed to list groups: {e}")
                self.app.render(f"ERROR: {parse_error_message(self.app, e)}\n")
                return

        else:
            group = db.get_group()
            if group is None or group.get('name') is None:
                self.app.log.debug('[group-command-ping] There is no active group.')
                self.app.render('There is no active group.\n')
                return

            group_id = group.get('id')

        command_request = GroupCommandRequest(command=DeviceCommandEnum.UPDATE_HEARTBEAT.name)
        try:
            response = command_client.run_group_command(enterprise_id, group_id, command_request)
        except ApiException as e:
            self.app.log.error(f"[group-command-ping] Failed to fire the ping group command: {e}")
            self.app.render(f"ERROR: {parse_error_message(self.app, e)}\n")
            return

        if not self.app.pargs.json:
            renderable = self._command_basic_response(response)
            self.app.render(renderable, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")
        else:
            renderable = self._command_basic_response(response, OutputFormat.JSON)
            self.app.render(renderable, format=OutputFormat.JSON.value)

    @ex(
        help='Lock group of devices',
        arguments=[
            (['-g', '--group'],
             {'help': 'Group name',
              'action': 'store',
              'dest': 'group'}),
            (['-j', '--json'],
             {'help': 'Render result in Json format',
              'action': 'store_true',
              'dest': 'json'}),
        ]
    )
    def lock(self):
        validate_creds_exists(self.app)
        db = DBWrapper(self.app.creds)
        command_client = APIClient(db.get_configure()).get_group_command_api_client()
        enterprise_id = db.get_enterprise_id()
        group_client = APIClient(db.get_configure()).get_group_api_client()

        if self.app.pargs.group:
            group_name = self.app.pargs.group
            kwargs = {'name': group_name}
            try:
                search_response = group_client.get_all_groups(enterprise_id, limit=1, offset=0, **kwargs)
                if not search_response.results or len(search_response.results) == 0:
                    self.app.log.debug(f'[group-command-lock] Group does not exist with name {group_name}')
                    self.app.render(f'Group does not exist with name {group_name}\n')
                    return
                response = search_response.results[0]
                group_id = response.id
            except ApiException as e:
                self.app.log.error(f"[group-command-lock] Failed to list groups: {e}")
                self.app.render(f"ERROR: {parse_error_message(self.app, e)}\n")
                return

        else:
            group = db.get_group()
            if group is None or group.get('name') is None:
                self.app.log.debug('[group-command-lock] There is no active group.')
                self.app.render('There is no active group.\n')
                return

            group_id = group.get('id')

        command_request = GroupCommandRequest(command=DeviceCommandEnum.LOCK.name)
        try:
            response = command_client.run_group_command(enterprise_id, group_id, command_request)
        except ApiException as e:
            self.app.log.error(f"[group-command-lock] Failed to fire the lock group command: {e}")
            self.app.render(f"ERROR: {parse_error_message(self.app, e)}\n")
            return

        if not self.app.pargs.json:
            renderable = self._command_basic_response(response)
            self.app.render(renderable, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")
        else:
            renderable = self._command_basic_response(response, OutputFormat.JSON)
            self.app.render(renderable, format=OutputFormat.JSON.value)

    @ex(
        help='Reboot group of devices',
        arguments=[
            (['-g', '--group'],
             {'help': 'Group name',
              'action': 'store',
              'dest': 'group'}),
            (['-j', '--json'],
             {'help': 'Render result in Json format',
              'action': 'store_true',
              'dest': 'json'}),
        ]
    )
    def reboot(self):
        validate_creds_exists(self.app)
        db = DBWrapper(self.app.creds)
        command_client = APIClient(db.get_configure()).get_group_command_api_client()
        enterprise_id = db.get_enterprise_id()
        group_client = APIClient(db.get_configure()).get_group_api_client()

        if self.app.pargs.group:
            group_name = self.app.pargs.group
            kwargs = {'name': group_name}
            try:
                search_response = group_client.get_all_groups(enterprise_id, limit=1, offset=0, **kwargs)
                if not search_response.results or len(search_response.results) == 0:
                    self.app.log.debug(f'[group-command-reboot] Group does not exist with name {group_name}')
                    self.app.render(f'Group does not exist with name {group_name}\n')
                    return
                response = search_response.results[0]
                group_id = response.id
            except ApiException as e:
                self.app.log.error(f"[group-command-reboot] Failed to list groups: {e}")
                self.app.render(f"ERROR: {parse_error_message(self.app, e)}\n")
                return

        else:
            group = db.get_group()
            if group is None or group.get('name') is None:
                self.app.log.debug('[group-command-reboot] There is no active group.')
                self.app.render('There is no active group.\n')
                return

            group_id = group.get('id')

        command_request = GroupCommandRequest(command=DeviceCommandEnum.REBOOT.name)
        try:
            response = command_client.run_group_command(enterprise_id, group_id, command_request)
        except ApiException as e:
            self.app.log.error(f"[group-command-reboot] Failed to fire the reboot group command: {e}")
            self.app.render(f"ERROR: {parse_error_message(self.app, e)}\n")
            return

        if not self.app.pargs.json:
            renderable = self._command_basic_response(response)
            self.app.render(renderable, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")
        else:
            renderable = self._command_basic_response(response, OutputFormat.JSON)
            self.app.render(renderable, format=OutputFormat.JSON.value)
