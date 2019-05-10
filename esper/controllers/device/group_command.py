from ast import literal_eval

from cement import ex, Controller
from cement.utils.version import get_version_banner
from crayons import white
from esperclient import GroupCommandRequest
from esperclient.rest import ApiException

from esper.controllers.enums import OutputFormat, DeviceCommandEnum
from esper.core.version import get_version
from esper.ext.api_client import APIClient
from esper.ext.db_wrapper import DBWrapper
from esper.ext.utils import validate_creds_exists

VERSION_BANNER = """
Command Line Tool for Esper SDK %s
%s
""" % (get_version(), get_version_banner())


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
            title = white("TITLE", bold=True)
            details = white("DETAILS", bold=True)
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
             {'help': 'Device command id',
              'action': 'store'}),
            (['-d', '--group'],
             {'help': 'Group id',
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

        if self.app.pargs.group:
            group_id = self.app.pargs.group
        else:
            group = db.get_group()
            if not group or not group.get('id'):
                self.app.log.info('Not set the active group.')
                return

            group_id = group.get('id')

        try:
            response = command_client.get_group_command(command_id, group_id, enterprise_id)
        except ApiException as e:
            self.app.log.debug(f"Failed to show details of group command: {e}")
            self.app.log.error(f"Failed to show details of group command, reason: {e.reason}")
            return

        if not self.app.pargs.json:
            renderable = self._command_basic_response(response)
            print(white(f"\tCOMMAND DETAILS of {response.command}", bold=True))
            self.app.render(renderable, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="fancy_grid")
        else:
            renderable = self._command_basic_response(response, OutputFormat.JSON)
            print(white(f"COMMAND DETAILS of {response.command}", bold=True))
            self.app.render(renderable, format=OutputFormat.JSON.value)

    @ex(
        help='Install application version',
        arguments=[
            (['-d', '--group'],
             {'help': 'Group id',
              'action': 'store',
              'dest': 'group'}),
            (['-v', '--version'],
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

        if self.app.pargs.group:
            group_id = self.app.pargs.group
        else:
            group = db.get_group()
            if not group or not group.get('id'):
                self.app.log.info('Not set the active group.')
                return

            group_id = group.get('id')

        version_id = self.app.pargs.version

        command_request = GroupCommandRequest(command_args={"app_version": version_id},
                                              command=DeviceCommandEnum.INSTALL.name)
        try:
            response = command_client.run_group_command(enterprise_id, group_id, command_request)
        except ApiException as e:
            self.app.log.debug(f"Failed to fire the install group command: {e}")
            self.app.log.error(f"Failed to install application, reason: {e.reason}")
            return

        if not self.app.pargs.json:
            renderable = self._command_basic_response(response)
            print(white(f"\tCOMMAND DETAILS of {response.command}", bold=True))
            self.app.render(renderable, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="fancy_grid")
        else:
            renderable = self._command_basic_response(response, OutputFormat.JSON)
            print(white(f"COMMAND DETAILS of {response.command}", bold=True))
            self.app.render(renderable, format=OutputFormat.JSON.value)

    @ex(
        help='Ping group of devices',
        arguments=[
            (['-d', '--group'],
             {'help': 'Group id',
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

        if self.app.pargs.group:
            group_id = self.app.pargs.group
        else:
            group = db.get_group()
            if not group or not group.get('id'):
                self.app.log.info('Not set the active group.')
                return

            group_id = group.get('id')

        command_request = GroupCommandRequest(command=DeviceCommandEnum.UPDATE_HEARTBEAT.name)
        try:
            response = command_client.run_group_command(enterprise_id, group_id, command_request)
        except ApiException as e:
            self.app.log.debug(f"Failed to fire the ping group command: {e}")
            self.app.log.error(f"Failed to ping the group of devices, reason: {e.reason}")
            return

        if not self.app.pargs.json:
            renderable = self._command_basic_response(response)
            print(white(f"\tCOMMAND DETAILS of {response.command}", bold=True))
            self.app.render(renderable, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="fancy_grid")
        else:
            renderable = self._command_basic_response(response, OutputFormat.JSON)
            print(white(f"COMMAND DETAILS of {response.command}", bold=True))
            self.app.render(renderable, format=OutputFormat.JSON.value)

    @ex(
        help='Lock group of devices',
        arguments=[
            (['-d', '--group'],
             {'help': 'Group id',
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

        if self.app.pargs.group:
            group_id = self.app.pargs.group
        else:
            group = db.get_group()
            if not group or not group.get('id'):
                self.app.log.info('Not set the active group.')
                return

            group_id = group.get('id')

        command_request = GroupCommandRequest(command=DeviceCommandEnum.LOCK.name)
        try:
            response = command_client.run_group_command(enterprise_id, group_id, command_request)
        except ApiException as e:
            self.app.log.debug(f"Failed to fire the lock group command: {e}")
            self.app.log.error(f"Failed to lock the group of devices, reason: {e.reason}")
            return

        if not self.app.pargs.json:
            renderable = self._command_basic_response(response)
            print(white(f"\tCOMMAND DETAILS of {response.command}", bold=True))
            self.app.render(renderable, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="fancy_grid")
        else:
            renderable = self._command_basic_response(response, OutputFormat.JSON)
            print(white(f"COMMAND DETAILS of {response.command}", bold=True))
            self.app.render(renderable, format=OutputFormat.JSON.value)

    @ex(
        help='Reboot group of devices',
        arguments=[
            (['-d', '--group'],
             {'help': 'Group id',
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

        if self.app.pargs.group:
            group_id = self.app.pargs.group
        else:
            group = db.get_group()
            if not group or not group.get('id'):
                self.app.log.info('Not set the active group.')
                return

            group_id = group.get('id')

        command_request = GroupCommandRequest(command=DeviceCommandEnum.REBOOT.name)
        try:
            response = command_client.run_group_command(enterprise_id, group_id, command_request)
        except ApiException as e:
            self.app.log.debug(f"Failed to fire the reboot group command: {e}")
            self.app.log.error(f"Failed to reboot the group of devices, reason: {e.reason}")
            return

        if not self.app.pargs.json:
            renderable = self._command_basic_response(response)
            print(white(f"\tCOMMAND DETAILS of {response.command}", bold=True))
            self.app.render(renderable, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="fancy_grid")
        else:
            renderable = self._command_basic_response(response, OutputFormat.JSON)
            print(white(f"COMMAND DETAILS of {response.command}", bold=True))
            self.app.render(renderable, format=OutputFormat.JSON.value)
