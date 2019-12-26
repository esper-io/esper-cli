from cement import ex, Controller
from esperclient import CommandRequest
from esperclient.rest import ApiException

from esper.controllers.enums import OutputFormat, DeviceCommandEnum
from esper.ext.api_client import APIClient
from esper.ext.db_wrapper import DBWrapper
from esper.ext.utils import validate_creds_exists, parse_error_message


class DeviceCommand(Controller):
    class Meta:
        label = 'device-command'

        # text displayed at the top of --help output
        description = 'Fire commands for device'

        # text displayed at the bottom of --help output
        epilog = 'Usage: espercli device-command'

        stacked_type = 'nested'
        stacked_on = 'base'

    def _command_basic_response(self, command, format=OutputFormat.TABULATED):
        valid_keys = ['id', 'command', 'state']

        if format == OutputFormat.TABULATED:
            title = "TITLE"
            details = "DETAILS"
            renderable = [{title: k, details: v} for k, v in command.to_dict().items() if k in valid_keys]
        else:
            renderable = {k: v for k, v in command.to_dict().items() if k in valid_keys}

        return renderable

    @ex(
        help='Show command details',
        arguments=[
            (['command_id'],
             {'help': 'Device command id',
              'action': 'store'}),
            (['-d', '--device'],
             {'help': 'Device name',
              'action': 'store',
              'dest': 'device'}),
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
        command_client = APIClient(db.get_configure()).get_command_api_client()
        enterprise_id = db.get_enterprise_id()
        device_client = APIClient(db.get_configure()).get_device_api_client()

        if self.app.pargs.device:
            device_name = self.app.pargs.device
            kwargs = {'name': device_name}
            try:
                search_response = device_client.get_all_devices(enterprise_id, limit=1, offset=0, **kwargs)
                if not search_response.results or len(search_response.results) == 0:
                    self.app.log.debug(f'[device-command-show] Device does not exist with name {device_name}')
                    self.app.render(f'Device does not exist with name {device_name}\n')
                    return
                response = search_response.results[0]
                device_id = response.id
            except ApiException as e:
                self.app.log.error(f"[device-command-show] Failed to list devices: {e}")
                self.app.render(f"ERROR: {parse_error_message(self.app, e)}\n")
                return
        else:
            device = db.get_device()
            if not device or not device.get('id'):
                self.app.log.debug('[device-command-show] There is no active device.')
                self.app.render('There is no active device.\n')
                return

            device_id = device.get('id')

        try:
            response = command_client.get_command(command_id, device_id, enterprise_id)
        except ApiException as e:
            self.app.log.error(f"[device-command-show] Failed to show details of command: {e}")
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
            (['-d', '--device'],
             {'help': 'Device name',
              'action': 'store',
              'dest': 'device'}),
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
        command_client = APIClient(db.get_configure()).get_command_api_client()
        enterprise_id = db.get_enterprise_id()
        device_client = APIClient(db.get_configure()).get_device_api_client()

        if self.app.pargs.device:
            device_name = self.app.pargs.device
            kwargs = {'name': device_name}
            try:
                search_response = device_client.get_all_devices(enterprise_id, limit=1, offset=0, **kwargs)
                if not search_response.results or len(search_response.results) == 0:
                    self.app.log.debug(f'[device-command-install] Device does not exist with name {device_name}')
                    self.app.render(f'Device does not exist with name {device_name}\n')
                    return
                response = search_response.results[0]
                device_id = response.id
            except ApiException as e:
                self.app.log.error(f"[device-command-install] Failed to list devices: {e}")
                self.app.render(f"ERROR: {parse_error_message(self.app, e)}\n")
                return
        else:
            device = db.get_device()
            if not device or not device.get('id'):
                self.app.log.debug('[device-command-install] There is no active device.')
                self.app.render('There is no active device.\n')
                return

            device_id = device.get('id')

        version_id = self.app.pargs.version
        command_request = CommandRequest(command_args={"app_version": version_id},
                                         command=DeviceCommandEnum.INSTALL.name)
        try:
            response = command_client.run_command(enterprise_id, device_id, command_request)
        except ApiException as e:
            self.app.log.error(f"[device-command-install] Failed to fire the install command: {e}")
            self.app.render(f"ERROR: {parse_error_message(self.app, e)}\n")
            return

        if not self.app.pargs.json:
            renderable = self._command_basic_response(response)
            self.app.render(renderable, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")
        else:
            renderable = self._command_basic_response(response, OutputFormat.JSON)
            self.app.render(renderable, format=OutputFormat.JSON.value)

    @ex(
        help='Uninstall application version',
        arguments=[
            (['-d', '--device'],
             {'help': 'Device name',
              'action': 'store',
              'dest': 'device'}),
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
    def uninstall(self):
        validate_creds_exists(self.app)
        db = DBWrapper(self.app.creds)
        command_client = APIClient(db.get_configure()).get_command_api_client()
        enterprise_id = db.get_enterprise_id()
        device_client = APIClient(db.get_configure()).get_device_api_client()

        if self.app.pargs.device:
            device_name = self.app.pargs.device
            kwargs = {'name': device_name}
            try:
                search_response = device_client.get_all_devices(enterprise_id, limit=1, offset=0, **kwargs)
                if not search_response.results or len(search_response.results) == 0:
                    self.app.log.debug(f'[device-command-uninstall] Device does not exist with name {device_name}')
                    self.app.render(f'Device does not exist with name {device_name}\n')
                    return
                response = search_response.results[0]
                device_id = response.id
            except ApiException as e:
                self.app.log.error(f"[device-command-uninstall] Failed to list devices: {e}")
                self.app.render(f"ERROR: {parse_error_message(self.app, e)}\n")
                return
        else:
            device = db.get_device()
            if not device or not device.get('id'):
                self.app.log.debug('[device-command-uninstall] There is no active device.')
                self.app.render('There is no active device.\n')
                return

            device_id = device.get('id')

        version_id = self.app.pargs.version
        command_request = CommandRequest(command_args={"app_version": version_id},
                                         command=DeviceCommandEnum.UNINSTALL.name)
        try:
            response = command_client.run_command(enterprise_id, device_id, command_request)
        except ApiException as e:
            self.app.log.error(f"[device-command-uninstall] Failed to fire the uninstall command: {e}")
            self.app.render(f"ERROR: {parse_error_message(self.app, e)}\n")
            return

        if not self.app.pargs.json:
            renderable = self._command_basic_response(response)
            self.app.render(renderable, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")
        else:
            renderable = self._command_basic_response(response, OutputFormat.JSON)
            self.app.render(renderable, format=OutputFormat.JSON.value)

    @ex(
        help='Ping a device',
        arguments=[
            (['-d', '--device'],
             {'help': 'Device name',
              'action': 'store',
              'dest': 'device'}),
            (['-j', '--json'],
             {'help': 'Render result in Json format',
              'action': 'store_true',
              'dest': 'json'}),
        ]
    )
    def ping(self):
        validate_creds_exists(self.app)
        db = DBWrapper(self.app.creds)
        command_client = APIClient(db.get_configure()).get_command_api_client()
        enterprise_id = db.get_enterprise_id()
        device_client = APIClient(db.get_configure()).get_device_api_client()

        if self.app.pargs.device:
            device_name = self.app.pargs.device
            kwargs = {'name': device_name}
            try:
                search_response = device_client.get_all_devices(enterprise_id, limit=1, offset=0, **kwargs)
                if not search_response.results or len(search_response.results) == 0:
                    self.app.log.debug(f'[device-command-ping] Device does not exist with name {device_name}')
                    self.app.render(f'Device does not exist with name {device_name}\n')
                    return
                response = search_response.results[0]
                device_id = response.id
            except ApiException as e:
                self.app.log.error(f"[device-command-ping] Failed to list devices: {e}")
                self.app.render(f"ERROR: {parse_error_message(self.app, e)}\n")
                return
        else:
            device = db.get_device()
            if not device or not device.get('id'):
                self.app.log.debug('[device-command-ping] There is no active device.')
                self.app.render('There is no active device.\n')
                return

            device_id = device.get('id')

        command_request = CommandRequest(command=DeviceCommandEnum.UPDATE_HEARTBEAT.name)
        try:
            response = command_client.run_command(enterprise_id, device_id, command_request)
        except ApiException as e:
            self.app.log.error(f"[device-command-ping] Failed to fire the ping command: {e}")
            self.app.render(f"ERROR: {parse_error_message(self.app, e)}\n")
            return

        if not self.app.pargs.json:
            renderable = self._command_basic_response(response)
            self.app.render(renderable, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")
        else:
            renderable = self._command_basic_response(response, OutputFormat.JSON)
            self.app.render(renderable, format=OutputFormat.JSON.value)

    @ex(
        help='Lock a device',
        arguments=[
            (['-d', '--device'],
             {'help': 'Device name',
              'action': 'store',
              'dest': 'device'}),
            (['-j', '--json'],
             {'help': 'Render result in Json format',
              'action': 'store_true',
              'dest': 'json'}),
        ]
    )
    def lock(self):
        validate_creds_exists(self.app)
        db = DBWrapper(self.app.creds)
        command_client = APIClient(db.get_configure()).get_command_api_client()
        enterprise_id = db.get_enterprise_id()
        device_client = APIClient(db.get_configure()).get_device_api_client()

        if self.app.pargs.device:
            device_name = self.app.pargs.device
            kwargs = {'name': device_name}
            try:
                search_response = device_client.get_all_devices(enterprise_id, limit=1, offset=0, **kwargs)
                if not search_response.results or len(search_response.results) == 0:
                    self.app.log.debug(f'[device-command-lock] Device does not exist with name {device_name}')
                    self.app.render(f'Device does not exist with name {device_name}\n')
                    return
                response = search_response.results[0]
                device_id = response.id
            except ApiException as e:
                self.app.log.error(f"[device-command-lock] Failed to list devices: {e}")
                self.app.render(f"ERROR: {parse_error_message(self.app, e)}\n")
                return
        else:
            device = db.get_device()
            if not device or not device.get('id'):
                self.app.log.debug('[device-command-lock] There is no active device.')
                self.app.render('There is no active device.\n')
                return

            device_id = device.get('id')

        command_request = CommandRequest(command=DeviceCommandEnum.LOCK.name)
        try:
            response = command_client.run_command(enterprise_id, device_id, command_request)
        except ApiException as e:
            self.app.log.error(f"[device-command-lock] Failed to fire the lock command: {e}")
            self.app.render(f"ERROR: {parse_error_message(self.app, e)}\n")
            return

        if not self.app.pargs.json:
            renderable = self._command_basic_response(response)
            self.app.render(renderable, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")
        else:
            renderable = self._command_basic_response(response, OutputFormat.JSON)
            self.app.render(renderable, format=OutputFormat.JSON.value)

    @ex(
        help='Reboot a device',
        arguments=[
            (['-d', '--device'],
             {'help': 'Device name',
              'action': 'store',
              'dest': 'device'}),
            (['-j', '--json'],
             {'help': 'Render result in Json format',
              'action': 'store_true',
              'dest': 'json'}),
        ]
    )
    def reboot(self):
        validate_creds_exists(self.app)
        db = DBWrapper(self.app.creds)
        command_client = APIClient(db.get_configure()).get_command_api_client()
        enterprise_id = db.get_enterprise_id()
        device_client = APIClient(db.get_configure()).get_device_api_client()

        if self.app.pargs.device:
            device_name = self.app.pargs.device
            kwargs = {'name': device_name}
            try:
                search_response = device_client.get_all_devices(enterprise_id, limit=1, offset=0, **kwargs)
                if not search_response.results or len(search_response.results) == 0:
                    self.app.log.debug(f'[device-command-reboot] Device does not exist with name {device_name}')
                    self.app.render(f'Device does not exist with name {device_name}\n')
                    return
                response = search_response.results[0]
                device_id = response.id
            except ApiException as e:
                self.app.log.error(f"[device-command-reboot] Failed to list devices: {e}")
                self.app.render(f"ERROR: {parse_error_message(self.app, e)}\n")
                return
        else:
            device = db.get_device()
            if not device or not device.get('id'):
                self.app.log.debug('[device-command-reboot] There is no active device.')
                self.app.render('There is no active device.\n')
                return

            device_id = device.get('id')

        command_request = CommandRequest(command=DeviceCommandEnum.REBOOT.name)
        try:
            response = command_client.run_command(enterprise_id, device_id, command_request)
        except ApiException as e:
            self.app.log.error(f"[device-command-reboot] Failed to fire the reboot command: {e}")
            self.app.render(f"ERROR: {parse_error_message(self.app, e)}\n")
            return

        if not self.app.pargs.json:
            renderable = self._command_basic_response(response)
            self.app.render(renderable, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")
        else:
            renderable = self._command_basic_response(response, OutputFormat.JSON)
            self.app.render(renderable, format=OutputFormat.JSON.value)

    @ex(
        help='Wipe a device',
        arguments=[
            (['-d', '--device'],
             {'help': 'Device name',
              'action': 'store',
              'dest': 'device'}),
            (['-e', '--exstorage'],
             {'help': 'External storage',
              'action': 'store_true',
              'dest': 'external_storage'}),
            (['-f', '--frp'],
             {'help': 'Factory reset production',
              'action': 'store_true',
              'dest': 'frp'}),
            (['-j', '--json'],
             {'help': 'Render result in Json format',
              'action': 'store_true',
              'dest': 'json'}),
        ]
    )
    def wipe(self):
        validate_creds_exists(self.app)
        db = DBWrapper(self.app.creds)
        command_client = APIClient(db.get_configure()).get_command_api_client()
        enterprise_id = db.get_enterprise_id()
        device_client = APIClient(db.get_configure()).get_device_api_client()

        if self.app.pargs.device:
            device_name = self.app.pargs.device
            kwargs = {'name': device_name}
            try:
                search_response = device_client.get_all_devices(enterprise_id, limit=1, offset=0, **kwargs)
                if not search_response.results or len(search_response.results) == 0:
                    self.app.log.debug(f'[device-command-wipe] Device does not exist with name {device_name}')
                    self.app.render(f'Device does not exist with name {device_name}\n')
                    return
                response = search_response.results[0]
                device_id = response.id
            except ApiException as e:
                self.app.log.error(f"[device-command-wipe] Failed to list devices: {e}")
                self.app.render(f"ERROR: {parse_error_message(self.app, e)}\n")
                return
        else:
            device = db.get_device()
            if not device or not device.get('id'):
                self.app.log.debug('[device-command-wipe] There is no active device.')
                self.app.render('There is no active device.\n')
                return

            device_id = device.get('id')

        external_storage = self.app.pargs.external_storage
        frp = self.app.pargs.frp

        if external_storage is None:
            self.app.log.info('[device-command-wipe] External storage value is empty')
            self.app.render('External storage value is empty\n')

        if frp is None:
            self.app.log.info('[device-command-wipe] Factory reset production value is empty')
            self.app.render('Factory reset production value is empty\n')

        command_request = CommandRequest(command_args={"wipe_external_storage": external_storage, 'wipe_FRP': frp},
                                         command=DeviceCommandEnum.WIPE.name)
        try:
            response = command_client.run_command(enterprise_id, device_id, command_request)
        except ApiException as e:
            self.app.log.error(f"[device-command-wipe] Failed to fire the wipe command: {e}")
            self.app.render(f"ERROR: {parse_error_message(self.app, e)}\n")
            return

        if not self.app.pargs.json:
            renderable = self._command_basic_response(response)
            self.app.render(renderable, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")
        else:
            renderable = self._command_basic_response(response, OutputFormat.JSON)
            self.app.render(renderable, format=OutputFormat.JSON.value)

    @ex(
        help='Clear app data',
        arguments=[
            (['-d', '--device'],
             {'help': 'Device name',
              'action': 'store',
              'dest': 'device'}),
            (['-P', '--package-name'],
             {'help': 'Application package name',
              'action': 'store',
              'dest': 'package_name'}),
            (['-j', '--json'],
             {'help': 'Render result in Json format',
              'action': 'store_true',
              'dest': 'json'}),
        ]
    )
    def clear_app_data(self):
        validate_creds_exists(self.app)
        db = DBWrapper(self.app.creds)
        command_client = APIClient(db.get_configure()).get_command_api_client()
        enterprise_id = db.get_enterprise_id()
        device_client = APIClient(db.get_configure()).get_device_api_client()

        if self.app.pargs.device:
            device_name = self.app.pargs.device
            kwargs = {'name': device_name}
            try:
                search_response = device_client.get_all_devices(enterprise_id, limit=1, offset=0, **kwargs)
                if not search_response.results or len(search_response.results) == 0:
                    self.app.log.debug(f'[device-command-clear-app-data] Device does not exist with name {device_name}')
                    self.app.render(f'Device does not exist with name {device_name}\n')
                    return
                response = search_response.results[0]
                device_id = response.id
            except ApiException as e:
                self.app.log.error(f"[device-command-clear-app-data] Failed to list devices: {e}")
                self.app.render(f"ERROR: {parse_error_message(self.app, e)}\n")
                return
        else:
            device = db.get_device()
            if not device or not device.get('id'):
                self.app.log.debug('[device-command-clear-app-data] There is no active device.')
                self.app.render('There is no active device.\n')
                return

            device_id = device.get('id')

        package_name = self.app.pargs.package_name
        if package_name is None:
            self.app.log.info('[device-command-clear-app-data] Package name is empty')
            self.app.render('Package name is empty\n')
            return

        command_request = CommandRequest(command_args={"package_name": package_name},
                                         command=DeviceCommandEnum.CLEAR_APP_DATA.name)
       
        try:
            response = command_client.run_command(enterprise_id, device_id, command_request)
        except ApiException as e:
            self.app.log.error(f"[device-command-clear-app-data] Failed to fire the CLEAR_APP_DATA command: {e}")
            self.app.render(f"ERROR: {parse_error_message(self.app, e)}\n")
            return

        if not self.app.pargs.json:
            renderable = self._command_basic_response(response)
            self.app.render(renderable, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")
        else:
            renderable = self._command_basic_response(response, OutputFormat.JSON)
            self.app.render(renderable, format=OutputFormat.JSON.value)
