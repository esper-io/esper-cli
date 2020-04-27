import time
import webbrowser

from cement import Controller, ex
from esperclient.rest import ApiException

from esper.controllers.enums import DeviceCommandState
from esper.ext.api_client import APIClient
from esper.ext.conference_api import create_room, get_demo_config, \
    get_command_status, send_conference_command, get_room_by_name
from esper.ext.db_wrapper import DBWrapper
from esper.ext.utils import validate_creds_exists, parse_error_message


class Conference(Controller):
    class Meta:
        label = 'conference'

        # text displayed at the top of --help output
        description = 'Commands to perform video conferencing with device'

        # text displayed at the bottom of --help output
        epilog = 'Usage: espercli conference'

        stacked_type = 'nested'
        stacked_on = 'base'

    @ex(
        help='Start a conference call',
        arguments=[
            (['-d', '--device'],
             {'help': 'Device name',
              'action': 'store',
              'dest': 'device_name'}),
        ]
    )
    def start(self):
        validate_creds_exists(self.app)
        db = DBWrapper(self.app.creds)
        environment = db.get_configure().get("environment")
        enterprise_id = db.get_enterprise_id()
        scapi_api_key = db.get_configure().get("api_key")
        device_client = APIClient(db.get_configure()).get_device_api_client()

        ok, response = get_demo_config(environment, scapi_api_key)
        if not ok:
            self.app.log.error(f"Response not OK. {response}")
            return
        dailyco_api_key = response.get("dailyco_api_key")

        device_name = self.app.pargs.device_name
        if not device_name:
            self.app.render(f'No device specified. Use the -d, --device option to specify a device\n')
            return

        self.app.render(f"Initiating conference call with device {device_name}...\n")

        # Fetch device id from device name supplied as parameter
        kwargs = {'name': device_name}
        try:
            response = device_client.get_all_devices(enterprise_id, limit=1, offset=0, **kwargs)
            if not response.results or len(response.results) == 0:
                self.app.log.debug(f'[device-show] Device does not exist with name {device_name}')
                self.app.render(f'Device does not exist with name {device_name}\n')
                return
            device_id = response.results[0].id
        except ApiException as e:
            self.app.log.error(f"[device-show] Failed to fetch telemetry info for device {device_name}: {e}")
            self.app.render(f"ERROR: {parse_error_message(self.app, e)}\n")
            return

        # 1. create a conference room
        room_name = f'{environment}-{device_name}'
        conference_duration = int(time.time()) + 1800  # 30 mins

        ok, response = create_room(room_name, conference_duration, dailyco_api_key)
        if not ok:
            self.app.log.debug(f"Response not OK. {response}")
            self.app.render(f"Failed to create conference room.\n")
            return

        # if HIPAA compliant, check custom room name
        # ok, response = get_room_by_name(room_name, dailyco_api_key)
        # if not ok:
        #     self.app.render(f"Room does not exists. Creating new room...\n")
        #     ok, response = create_room(room_name, conference_duration, dailyco_api_key)
        #     if not ok:
        #         self.app.log.debug(f"Response not OK. {response}")
        #         self.app.render(f"Failed to create conference room.\n")
        #         return

        room_url = response.get("url")

        # 2. Send command to device
        data = {
            "room_url": room_url
        }

        ok, response = send_conference_command(environment, enterprise_id, device_id, scapi_api_key, data)
        if not ok:
            self.app.log.debug(f"Response not OK. {response}")
            self.app.render(f"Failed to communicate with device.\n")
            return

        self.app.render(f"Awaiting device acknowledgement....\n")

        command_pending = True
        command_id = response.get("id")
        while command_pending:
            ok, response = get_command_status(environment, enterprise_id, scapi_api_key, command_id)
            if not ok:
                self.app.log.debug(f"Response not OK. {response}")
                self.app.render(f"Failed to communicate with device.\n")
                return

            command_state = response.get("results")[0].get("state")
            if command_state == DeviceCommandState.SUCCESS.value:
                self.app.render(f"Conference room created at : {room_url}\n")
                self.app.render("You can also visit the link in the browser of your choice\n")
                self.app.render(f"Connection successful with device, joining meeting...\n")
                webbrowser.open(room_url)
                command_pending = False
            elif command_state == DeviceCommandState.TIMEOUT.value:
                self.app.render(f"Connection timeout. Please check ensure device is connected to internet\n")
                command_pending = False
            time.sleep(2)
