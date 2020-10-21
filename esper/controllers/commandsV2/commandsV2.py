import json

from cement import ex, Controller
from esperclient import V0CommandRequest
from esperclient.rest import ApiException

from esper.controllers.enums import (
    OutputFormat, 
    CommandEnum, 
    CommandState,
    CommandScheduleEnum,
    CommandRequestTypeEnum,
    CommandDeviceTypeEnum,
    WeekDays
)
from esper.ext.api_client import APIClient
from esper.ext.db_wrapper import DBWrapper
from esper.ext.utils import validate_creds_exists, parse_error_message


class CommandsV2(Controller):
    class Meta:
        label = 'commandsV2'

        # text displayed at the top of --help output
        description = 'Fire commands to devices or groups'

        # text displayed at the bottom of --help output
        epilog = 'Usage: espercli commandsV2'

        stacked_type = 'nested'
        stacked_on = 'base'

    def _request_basic_response(self, request, format=OutputFormat.TABULATED):
        if request.schedule_args is None:
                schedule_args = None
        else:
            start_datetime = request.schedule_args.start_datetime
            end_datetime = request.schedule_args.end_datetime
            schedule_args = {
                "name": request.schedule_args.name,
                "start_datetime": str(start_datetime),
                "end_datetime": str(end_datetime),
                "time_type": request.schedule_args.time_type,
                "window_start_time": str(request.schedule_args.window_start_time),
                "window_end_time": str(request.schedule_args.window_end_time),
                "days": request.schedule_args.days
            }

        command_args = {
            "app_state": request.command_args.app_state,
            "app_version": request.command_args.app_version,
            "custom_settings_config": request.command_args.custom_settings_config,
            "device_alias_name": request.command_args.device_alias_name,
            "message": request.command_args.message,
            "package_name": request.command_args.package_name,
            "policy_url": request.command_args.policy_url,
            "state": request.command_args.state,
            "wifi_access_points": request.command_args.wifi_access_points
        }
        if len(request.status) != 0:
            state = request.status[0].state
        else:
            state = None
        if format == OutputFormat.TABULATED:
            title = "TITLE"
            details = "DETAILS"
            renderable = [
                {title: 'Id', details: request.id},
                {title: 'Command', details: request.command},
                {title: 'Command Args', details: request.command_args},
                {title: 'Command Type', details: request.command_type},
                {title: 'Devices', details: request.devices},
                {title: 'Groups', details: request.groups},
                {title: 'Device Type', details: request.device_type},
                {title: 'Status', details: state},
                {title: 'Issued by', details: request.issued_by},
                {title: 'Schedule', details: request.schedule},
                {title: 'Schedule Args', details: schedule_args},
                {title: 'Created On', details: request.created_on}
            ]
            
        else:
            renderable = {
                    "id": request.id,
                    "command": request.command,
                    "command_type": request.command_type,
                    "command_args": command_args,
                    "devices": request.devices,
                    "groups": request.groups,
                    "device_type": request.device_type,
                    "status": str(request.status),
                    "issued_by": request.issued_by,
                    "schedule": request.schedule,
                    "schedule_args": schedule_args,
                    "created_on": str(request.created_on),
                }

        return renderable
        

    
    def _status_basic_response(self, status, limit, format=OutputFormat.TABULATED):
        self.app.render(f"Total Number of Statuses: {status.count}\n")
        if not self.app.pargs.json:
            count_stat = 0
            statuses = []

            label = {
                'id': "STATUS ID",
                'device': "DEVICE ID",
                'state': "STATE",
                'reason': "REASON"
            }

            for status in status.results:
                if(count_stat < limit):
                    device_id =  status.device.split('/')[-2]
                    statuses.append(
                        {
                            label['id']: status.id,
                            label['device']: device_id,
                            label['state']: status.state,
                            label['reason']: status.reason,
                        }
                    )
                    count_stat+=1
            self.app.render(statuses, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")
        else:
            count_stat = 0
            statuses = []
            for status in status.results:
                if(count_stat < limit):
                    device_id =  status.device.split('/')[-2]
                    statuses.append(
                        {
                            'id': status.id,
                            'request_id': status.request,
                            'device_id': device_id,
                            'state': status.state,
                            'reason': status.reason,
                            'created_on': str(status.created_on),
                            'updated_on': str(status.updated_on)
                        }
                    )
                    count_stat+=1
            self.app.render(statuses, format=OutputFormat.JSON.value)


    @ex(
        help='List command requests',
        arguments=[
            (['-ct', '--command_type'],
             {'help': 'Filter by type of command request',
              'action': 'store',
              'choices': CommandRequestTypeEnum.choice_list_lower(),
              'dest': 'command_type'}),
            (['-d', '--device'],
             {'help': 'Filter by device name.',
              'action': 'store',
              'dest': 'device'}),
            (['-dt', '--device_type'],
             {'help': 'Filter by device type.',
              'action': 'store',
              'choices': CommandDeviceTypeEnum.choice_list_lower(),
              'dest': 'device_type'}),
            (['-c', '--command'],
             {'help': 'Filter by command name.',
              'action': 'store',
              'choices': CommandEnum.choice_list_lower(),
              'dest': 'command'}),
            (['-l', '--limit'],
             {'help': 'No. of results',
              'action': 'store',
              'default': 10,
              'type': int,
              'dest': 'limit'}),
            (['-j', '--json'],
             {'help': 'Render result in Json format',
              'action': 'store_true',
              'dest': 'json'}),
        ]
    )
    def list(self):
        validate_creds_exists(self.app)
        db = DBWrapper(self.app.creds)
        device_client = APIClient(db.get_configure()).get_device_api_client()
        commandsV2_client = APIClient(db.get_configure()).get_commandsV2_api_client()
        enterprise_id = db.get_enterprise_id()

        kwargs = {}
        if self.app.pargs.command_type:
            kwargs['command_type'] = self.app.pargs.command_type.upper()

        if self.app.pargs.device:
            device_name = self.app.pargs.device
            kw = {'name': device_name}
            try:
                search_response = device_client.get_all_devices(enterprise_id, limit=1, offset=0, **kw)
                if not search_response.results or len(search_response.results) == 0:
                    self.app.log.debug(f"[commandsV2-list] Device does not exist with name {device_name}")
                    self.app.render(f'Device does not exist with name {device_name}\n')
                    return
                response = search_response.results[0]
                kwargs['devices'] = response.id
            except ApiException as e:
                self.app.log.error(f"[commandsV2-list] Failed to list devices: {e}")
                self.app.render(f"ERROR: {parse_error_message(self.app, e)}")
                return

        if self.app.pargs.device_type:
            kwargs['device_type'] = self.app.pargs.device_type

        if self.app.pargs.command:
            kwargs['command'] = self.app.pargs.command.upper()
            
        try:
            response = commandsV2_client.list_command_request(enterprise_id, **kwargs)
        except ApiException as e:
            self.app.log.error(f"[commandsV2-list] Failed to list command requests: {e}")
            self.app.render(f"ERROR: {parse_error_message(self.app, e)}\n")
            return
        
        limit = self.app.pargs.limit

        self.app.render(f"Total Number of Command Requests: {response.count}\n")
        if not self.app.pargs.json:
            commandreqs = []
            count_req = 0
            label = {
                'id': "RQUEST ID",
                'command': "COMMAND",
                'issued_by': "ISSUED BY",
                'command_type': "COMMAND TYPE",
                'created_on': "CREATED ON"
            }

            for commandreq in response.results:
                if(count_req < limit):
                    issued_by = commandreq.issued_by.replace("'",'"')
                    issued_by_json = json.loads(issued_by)
                    commandreqs.append(
                        {
                            label['id']: commandreq.id,
                            label['command']: commandreq.command,
                            label['issued_by']: issued_by_json["username"],
                            label['command_type']: commandreq.command_type,
                            label['created_on']: commandreq.created_on
                        }
                    )
                    count_req+=1
            self.app.render(commandreqs, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")
        else:
            commandreqs = []
            count_req = 0
            for commandreq in response.results:
                if(count_req < limit):
                    if commandreq.schedule_args is None:
                        schedule_args = None
                    else:
                        schedule_args = {
                            "name": commandreq.schedule_args.name,
                            "start_datetime": str(commandreq.schedule_args.start_datetime),
                            "end_datetime": str(commandreq.schedule_args.end_datetime),
                            "time_type": commandreq.schedule_args.time_type,
                            "window_start_time": str(commandreq.schedule_args.window_start_time),
                            "window_end_time": str(commandreq.schedule_args.window_end_time),
                            "days": commandreq.schedule_args.days
                        }
                    command_args = {
                        "app_state": commandreq.command_args.app_state,
                        "app_version": commandreq.command_args.app_version,
                        "custom_settings_config": commandreq.command_args.custom_settings_config,
                        "device_alias_name": commandreq.command_args.device_alias_name,
                        "message": commandreq.command_args.message,
                        "package_name": commandreq.command_args.package_name,
                        "policy_url": commandreq.command_args.policy_url,
                        "state": commandreq.command_args.state,
                        "wifi_access_points": commandreq.command_args.wifi_access_points
                    }

                    commandreqs.append(
                        {   
                            'id': commandreq.id,
                            'command': commandreq.command,
                            'command_type': commandreq.command_type,
                            'issued_by': commandreq.issued_by,
                            "devices": commandreq.devices,
                            "device_type": commandreq.device_type,
                            "groups": commandreq.groups,
                            "command_args": command_args,
                            "schedule": commandreq.schedule,
                            "schedule_args": schedule_args,
                            "created_on": str(commandreq.created_on),
                            "status": str(commandreq.status)
                        }
                    )
                    count_req+=1
            self.app.render(commandreqs, format=OutputFormat.JSON.value)
        
        


    @ex(
        help='Show command request status',
        arguments=[
            (['-r', '--request'],
             {'help': 'Command Request Id',
              'action': 'store',
              'dest': 'request'}),
            (['-d', '--device'],
             {'help': 'Filter by device id',
              'action': 'store',
              'dest': 'device'}),
            (['-s', '--state'],
             {'help': 'Filter by command state.',
              'choices': CommandState.choice_list_lower(),
              'action': 'store',
              'dest': 'state'}),
            (['-l', '--limit'],
             {'help': 'No. of results',
              'action': 'store',
              'default': 10,
              'type': int,
              'dest': 'limit'}),
            (['-j', '--json'],
             {'help': 'Render result in Json format',
              'action': 'store_true',
              'dest': 'json'}),
        ]
    )
    def status(self):
        validate_creds_exists(self.app)
        db = DBWrapper(self.app.creds)
        device_client = APIClient(db.get_configure()).get_device_api_client()
        commandsV2_client = APIClient(db.get_configure()).get_commandsV2_api_client()
        enterprise_id = db.get_enterprise_id()

        if self.app.pargs.request:
            request_id = self.app.pargs.request
        else:
            self.app.log.debug('request id is not given')
            self.app.render('request id is not given\n')
            return 

        kwargs = {}
        if self.app.pargs.device:
            device_name = self.app.pargs.device
            kw = {'name': device_name}
            try:
                search_response = device_client.get_all_devices(enterprise_id, limit=1, offset=0, **kw)
                if not search_response.results or len(search_response.results) == 0:
                    self.app.log.debug(f'[commandsV2-status] Device does not exist with name {device_name}')
                    self.app.render(f'Device does not exist with name {device_name}\n')
                    return
                response = search_response.results[0]
                kwargs['device'] = response.id
            except ApiException as e:
                self.app.log.error(f"[commandsV2-status] Failed to list devices: {e}")
                self.app.render(f"ERROR: {parse_error_message(self.app, e)}")
                return

        if self.app.pargs.state:
            kwargs['state'] = CommandState[self.app.pargs.state.upper()].value

        try:
            response = commandsV2_client.get_command_request_status(enterprise_id, request_id, **kwargs)     
        except ApiException as e:
            self.app.log.error(f"[commandsV2-status] Failed to show status for id {request_id}: {e}")
            self.app.render(f"ERROR: {parse_error_message(self.app, e)}\n")
            return

        limit = self.app.pargs.limit
        
        if not self.app.pargs.json:
            renderable = self._status_basic_response(response, limit)
            self.app.render(renderable, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")
        else:
            renderable = self._status_basic_response(response, limit, OutputFormat.JSON)
            self.app.render(renderable, format=OutputFormat.JSON.value)
        


        
    @ex(
        help='Show device command history',
        arguments=[
            (['-d', '--device'],
             {'help': 'Device id',
              'action': 'store',
              'dest': 'device'}),
            (['-s', '--state'],
             {'help': 'Filter by command state.',
              'choices': CommandState.choice_list_lower(),
              'action': 'store',
              'dest': 'state'}),
            (['-l', '--limit'],
             {'help': 'No. of results',
              'action': 'store',
              'default': 10,
              'type': int,
              'dest': 'limit'}),
            (['-j', '--json'],
             {'help': 'Render result in Json format',
              'action': 'store_true',
              'dest': 'json'}),
        ]
    )
    def history(self):
        validate_creds_exists(self.app)
        db = DBWrapper(self.app.creds)
        commandsV2_client = APIClient(db.get_configure()).get_commandsV2_api_client()
        device_client = APIClient(db.get_configure()).get_device_api_client()
        enterprise_id = db.get_enterprise_id()

        if self.app.pargs.device:
            device_name = self.app.pargs.device
            kw = {'name': device_name}
            try:
                search_response = device_client.get_all_devices(enterprise_id, limit=1, offset=0, **kw)
                if not search_response.results or len(search_response.results) == 0:
                    self.app.log.debug(f'[commandsV2-history] Device does not exist with name {device_name}')
                    self.app.render(f'Device does not exist with name {device_name}\n')
                    return
                response = search_response.results[0]
                device_id = response.id
            except ApiException as e:
                self.app.log.error(f"[commandsV2-history] Failed to list devices: {e}")
                self.app.render(f"ERROR: {parse_error_message(self.app, e)}")
                return
        else:
            self.app.log.debug('Device name is not given')
            self.app.render('Device name is not given\n')
            return 

        kwargs = {}
        if self.app.pargs.state:
            kwargs['state'] = CommandState[self.app.pargs.state.upper()].value

        try:
            response = commandsV2_client.get_device_command_history(enterprise_id, device_id, **kwargs)
        except ApiException as e:
            self.app.log.error(f"[commandsV2-history] Failed to show history for id {device_id}: {e}")
            self.app.render(f"ERROR: {parse_error_message(self.app, e)}\n")
            return
        
        limit = self.app.pargs.limit

        if not self.app.pargs.json:
            renderable = self._status_basic_response(response, limit)
            self.app.render(renderable, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")
        else:
            renderable = self._status_basic_response(response, limit, OutputFormat.JSON)
            self.app.render(renderable, format=OutputFormat.JSON.value)


        
    @ex(
        help='Fire commands to devices and groups',
        arguments=[
            (['-ct', '--command_type'],
             {'help': 'Command type.',
              'action': 'store',
              'choices': ['device', 'group', 'dynamic'],
              'dest': 'command_type'}),
            (['-d', '--devices'],
             {'help': 'List of device names, space separated.',
              'nargs': "*",
              'type': str,
              'dest': 'devices'}),
            (['-g', '--groups'],
             {'help': 'List of group ids, space separated.',
              'nargs': "*",
              'type': str,
              'dest': 'groups'}),
            (['-dt', '--device_type'],
             {'help': 'Device type.',
              'action': 'store',
              'choices': ['active', 'inactive', 'all'],
              'default': 'active',
              'dest': 'device_type'}),
            (['-c', '--command'],
             {'help': 'Command name.',
              'action': 'store',
              'choices': CommandEnum.choice_list_lower(),
              'dest': 'command'}),
            (['-s', '--schedule'],
             {'help': 'Schedule type.',
              'action': 'store',
              'choices': ['immediate', 'window', 'recurring'],
              'default': 'immediate',
              'dest': 'schedule'}),
            (['-sn', '--schedule_name'],
             {'help': 'Schedule name.',
              'action': 'store',
              'dest': 'schedule_name'}),
            (['-st', '--start'],
             {'help': 'Start date-time.',
              'action': 'store',
              'dest': 'start_datetime'}),
            (['-en', '--end'],
             {'help': 'End date-time.',
              'action': 'store',
              'dest': 'end_datetime'}),
            (['-tt', '--time_type'],
             {'help': 'Time type.',
              'action': 'store',
              'choices': ['console', 'device'],
              'default': 'console',
              'dest': 'time_type'}),
            (['-ws', '--window_start'],
             {'help': 'Window start time.',
              'action': 'store',
              'dest': 'window_start_time'}),
            (['-we', '--window_end'],
             {'help': 'Window end time.',
              'action': 'store',
              'dest': 'window_end_time'}),
            (['-dy', '--days'],
             {'help': 'List of days.',
              'nargs': "*",
              'type': str,
              'choices': ['monday', 'tuesday', 'wednesday', 'thursday', 'friday', 'saturday', 'sunday', 'all'],
              'default': ['all'],
              'dest': 'days'}),
            (['-as', '--app_state'],
             {'help': 'App state',
              'action': 'store',
              'dest': 'app_state'}),
            (['-av', '--app_version'],
             {'help': 'App version',
              'action': 'store',
              'dest': 'app_version'}),
            (['-cs', '--custom_config'],
             {'help': 'Custom settings config',
              'action': 'store',
              'dest': 'custom_settings_config'}),
            (['-dv', '--device_alias'],
             {'help': 'Device alias name',
              'action': 'store',
              'dest': 'device_alias_name'}),
            (['-m', '--message'],
             {'help': 'Message',
              'action': 'store',
              'dest': 'message'}),
            (['-pk', '--package_name'],
             {'help': 'Package name',
              'action': 'store',
              'dest': 'package_name'}),
            (['-po', '--policy_url'],
             {'help': 'Policy URL',
              'action': 'store',
              'dest': 'policy_url'}),
            (['-se', '--state'],
             {'help': 'State',
              'action': 'store',
              'dest': 'state'}),
            (['-wap', '--wifi_access_points'],
             {'help': 'Wifi access points',
              'action': 'store',
              'dest': 'wifi_access_points'}),
            (['-j', '--json'],
             {'help': 'Render result in Json format',
              'action': 'store_true',
              'dest': 'json'}),
        ]
    )    
    def command(self):
        validate_creds_exists(self.app)
        db = DBWrapper(self.app.creds)
        commandsV2_client = APIClient(db.get_configure()).get_commandsV2_api_client()
        enterprise_id = db.get_enterprise_id()

        devices = self.app.pargs.devices
        device_type = self.app.pargs.device_type
        groups = self.app.pargs.groups
        schedule = self.app.pargs.schedule.upper()

        if self.app.pargs.command:
            command = self.app.pargs.command.upper()
        else:
            self.app.log.debug('[commandsV2-command] command cannot be empty.')
            self.app.render('command cannot be empty.')
            return

        if self.app.pargs.command_type:
            command_type = self.app.pargs.command_type.upper()
        else:
            self.app.log.debug('[commandsV2-command] command type cannot be empty.')
            self.app.render('command type cannot be empty.')
            return

        if command_type == CommandRequestTypeEnum.DEVICE.name or command_type == CommandRequestTypeEnum.DYNAMIC.name:
            if not devices or len(devices) == 0:
                self.app.log.debug('[commandsV2-command] devices cannot be empty.')
                self.app.render('devices cannot be empty.')
                return

        if command_type == CommandRequestTypeEnum.GROUP.name:
            if not groups or len(groups) == 0:
                self.app.log.debug('[commandsV2-command] groups cannot be empty.')
                self.app.render('groups cannot be empty.')
                return

        
        schedule_name = self.app.pargs.schedule_name
        start_datetime = self.app.pargs.start_datetime
        end_datetime = self.app.pargs.end_datetime
        window_start_time = self.app.pargs.window_start_time
        window_end_time = self.app.pargs.window_end_time
        time_type = self.app.pargs.time_type
        req_days = self.app.pargs.days
        days = []

        for day in req_days:
            if day == 'all':
                days.append(WeekDays.ALL_DAYS.value)
            else:
                day = WeekDays[day.upper()].value
                days.append(day)
            
        if schedule_name is None:
            schedule_args = None 
        else:
            schedule_args = {
                'name': schedule_name,
                'start_datetime': start_datetime,
                'end_datetime': end_datetime,
                'window_start_time': window_start_time,
                'window_end_time': window_end_time,
                'time_type': time_type,
                'days': days
            }  
    
        device_client = APIClient(db.get_configure()).get_device_api_client()
        device_ids = []
        for device_name in devices:
            kwargs = {'name': device_name}
            try:
                search_response = device_client.get_all_devices(enterprise_id, limit=1, offset=0, **kwargs)
                if not search_response.results or len(search_response.results) == 0:
                    self.app.log.debug(f'[commandsV2-command] Device does not exist with name {device_name}')
                    self.app.render(f'Device does not exist with name {device_name}')
                    return
                response = search_response.results[0]
                device_ids.append(response.id)
            except ApiException as e:
                self.app.log.error(f"Failed to list devices: {e}")
                self.app.render(f"ERROR: {parse_error_message(self.app, e)}")
                return

        if groups:
            group_client = APIClient(db.get_configure()).get_group_api_client()
            group_ids = []
            for group_id in groups:
                try:
                    response = group_client.get_group_by_id(group_id, enterprise_id)
                    group_ids.append(group_id)
                except ApiException as e:
                    self.app.log.error(f"[] Group does not exist with id {group_id}: {e}")
                    self.app.render(f"ERROR: {parse_error_message(self.app, e)}")
                    return

        command_args_ = {
            "app_state": self.app.pargs.app_state,
            "app_version": self.app.pargs.app_version,
            "custom_settings_config": self.app.pargs.custom_settings_config,
            "device_alias_name": self.app.pargs.device_alias_name,
            "message": self.app.pargs.message,
            "package_name": self.app.pargs.package_name,
            "policy_url": self.app.pargs.policy_url,
            "state": self.app.pargs.state,
            "wifi_access_points": self.app.pargs.wifi_access_points
        }
        command_args = {}
        for key, value in command_args_.items():
            if(value is not None):
                command_args[key] = value

        command_request = V0CommandRequest( command_type=command_type,
                                            devices=device_ids,
                                            device_type=device_type,
                                            command=command,
                                            command_args=None, 
                                            schedule=schedule,
                                            schedule_args=schedule_args
                                        )

        try:
            response = commandsV2_client.create_command(enterprise_id, command_request)
        except ApiException as e:
            self.app.log.error(f"[commandsV2-command] Failed to fire the command {command}: {e}")
            self.app.render(f"ERROR: {parse_error_message(self.app, e)}\n")
            return

        if not self.app.pargs.json:
            renderable = self._request_basic_response(response)
            self.app.render(renderable, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")
        else:
            renderable = self._request_basic_response(response, OutputFormat.JSON)
            self.app.render(renderable, format=OutputFormat.JSON.value)
    


