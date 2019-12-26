from cement import Controller, ex
from esperclient.rest import ApiException

from esper.controllers.enums import OutputFormat
from esper.ext.api_client import APIClient
from esper.ext.db_wrapper import DBWrapper
from esper.ext.telemetry_api import get_telemetry_url
from esper.ext.utils import validate_creds_exists, parse_error_message

from datetime import datetime, timedelta
import requests


class Telemetry(Controller):
    class Meta:
        label = 'telemetry'

        # text displayed at the top of --help output
        description = 'Telemetry commands'

        # text displayed at the bottom of --help output
        epilog = 'Usage: espercli telemetry'

        stacked_type = 'nested'
        stacked_on = 'base'

    @ex(
        help='Get telemetry data for a device',
        arguments=[
            (['-d', '--device'],
             {'help': 'Device name',
              'action': 'store',
              'dest': 'device_name'}),
            (['-m', '--metric'],
             {'help': 'Metric name for telemetry data',
              'action': 'store',
              'dest': 'metric'}),
            (['-f', '--from'],
             {'help': 'Start date time of telemetry data',
              'action': 'store',
              'dest': 'from_time',
              'default': datetime.now() - timedelta(days=2)}),
            (['-p', '--period'],
             {'help': '',
              'action': 'store',
              'dest': 'period',
              'default': 'hour',
              'choices': ['hour', 'month', 'day']}),
            (['-s', '--statistic'],
             {'help': 'Statistic function',
              'action': 'store',
              'dest': 'statistic',
              'default': 'avg',
              'choices': ['avg', 'sum', 'count']}),
            (['-l', '--last'],
             {'help': 'Relative time from now. Use -n for n hour\'s since or n days since',
              'action': 'store',
              'dest': 'last',}),
            (['-t', '--to'],
             {'help': 'End date time of telemetry data',
              'action': 'store',
              'dest': 'to_time',
              'default': datetime.now()}),
        ]
    )
    def get_data(self):
        validate_creds_exists(self.app)
        db = DBWrapper(self.app.creds)
        environment = db.get_configure().get("environment")
        enterprise_id = db.get_enterprise_id()
        device_client = APIClient(db.get_configure()).get_device_api_client()

        device_name = self.app.pargs.device_name
        if not device_name:
            self.app.render(f'No device specified. Use the -d, --device option to specify a device\n')
            return
        kwargs = {'name': device_name}

        # Fetch device id from device name supplied as parameter
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

        last = self.app.pargs.last
        period = self.app.pargs.period
        statistic = self.app.pargs.statistic

        from_time = self.app.pargs.from_time
        if last:
            current_date = datetime.now()
            if period == 'hour':
                from_time = current_date - timedelta(hours=int(last))
            elif period == 'day':
                from_time = current_date - timedelta(days=int(last))
            else:
                from_time = current_date.replace(month=datetime.now().date().month - int(last)) #- timedelta(days=int(last))
        from_time = str(from_time).replace(' ', 'T')
        if '.' not in from_time:
            from_time = from_time + '.0000Z'
        if 'Z' not in from_time:
            from_time = from_time + 'Z'
        to_time = self.app.pargs.to_time
        if last:
            to_time = datetime.now()
        to_time = str(to_time).replace(' ', 'T')
        if '.' not in to_time:
            to_time = to_time + '.0000Z'
        if 'Z' not in to_time:
            to_time = to_time + 'Z'

        if '-' not in self.app.pargs.metric:
            self.app.render("ERROR: Metric must be of format {category}-{metric name}\n")
            return

        category, metric = self.app.pargs.metric.split('-')
        # Calling Telemetry Graphs API
        url = get_telemetry_url(environment, enterprise_id, device_id, category, metric, from_time, to_time, period,
                                statistic)

        api_key = db.get_configure().get("api_key")
        response = requests.get(
            url,
            headers={
                'Authorization': f'Bearer {api_key}'
            }
        )

        response_json = response.json()
        if response.status_code != 200:
            if response_json.get('meta', {}).get('non_field_errors'):
                error_list = response_json.get('meta', {}).get('non_field_errors')
                self.app.render(f"ERRORS: {error_list}\n")
            elif response_json.get('errors'):
                self.app.render(f"ERRORS: {response_json.get('errors')}\n")
            else:
                #print(response_json, from_time, to_time)
                self.app.render(f"ERROR: Unknown error occurred\n")
            return

        data = response.json().get('data', {})

        # Rendering table with populated values
        label = {
            'time': "Time",
            'value': "Value"
        }

        render_data = []
        for d in data:
            render_data.append(
                {
                    label['time']: d['x'],
                    label['value']: d['y']
                }
            )
        if len(render_data) == 0:
            self.app.render(f"No telemetry data for device {device_name} found between time range\n")
            return
        self.app.render(f"Telemetry data for device {device_name}")
        self.app.render(render_data, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")


