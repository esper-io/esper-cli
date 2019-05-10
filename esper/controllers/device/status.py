from ast import literal_eval

from cement import Controller, ex
from cement.utils.version import get_version_banner
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


class DeviceStatus(Controller):
    class Meta:
        label = 'status'

        # text displayed at the top of --help output
        description = 'Device status commands'

        # text displayed at the bottom of --help output
        epilog = 'Usage: espercli status'

        stacked_type = 'nested'
        stacked_on = 'base'

    @ex(
        help='Latest device status',
        arguments=[
            (['-d', '--device'],
             {'help': 'Device id',
              'action': 'store',
              'dest': 'device'}),
            (['-j', '--json'],
             {'help': 'Render result in Json format',
              'action': 'store_true',
              'dest': 'json'})
        ]
    )
    def latest(self):
        """Command to list status"""
        validate_creds_exists(self.app)
        db = DBWrapper(self.app.creds)
        device_client = APIClient(db.get_configure()).get_device_api_client()
        enterprise_id = db.get_enterprise_id()

        if self.app.pargs.device:
            device_id = self.app.pargs.device
        else:
            device = db.get_device()
            if not device or not device.get('id'):
                self.app.log.info('Not set the active device.')
                return

            device_id = device.get('id')

        try:
            response = device_client.get_device_event(enterprise_id, device_id, latest_event=1)
        except ApiException as e:
            self.app.log.debug(f"Failed to get latest device status: {e}")
            self.app.log.error(f"Failed to get latest device status, reason: {e.reason}")
            return

        battery_level = None
        battery_temp = None
        data_download = None
        data_upload = None
        memory_storage = None
        memory_ram = None
        link_speed = None
        signal_strength = None

        if response.results and len(response.results) > 0:
            data = literal_eval(response.results[0].data)
            if data.get("powerManagementEvent") and data.get("powerManagementEvent").get("batteryStatus"):
                if data.get("powerManagementEvent").get("batteryStatus").get("batteryLevel"):
                    battery_level = data.get("powerManagementEvent").get("batteryStatus").get("batteryLevel")

                if data.get("powerManagementEvent").get("batteryStatus").get("batteryTemperature"):
                    battery_temp = data.get("powerManagementEvent").get("batteryStatus").get("batteryTemperature")

            if data.get("dataUsageStats"):
                if data.get("dataUsageStats").get("totalDataDownload"):
                    data_download = data.get("dataUsageStats").get("totalDataDownload")

                if data.get("dataUsageStats").get("totalDataUpload"):
                    data_upload = data.get("dataUsageStats").get("totalDataUpload")

            if data.get("memoryEvents") and len(data.get("memoryEvents")) > 1 and \
                    data.get("memoryEvents")[1].get("countInMb"):
                memory_storage = data.get("memoryEvents")[1].get("countInMb")

            if data.get("memoryEvents") and len(data.get("memoryEvents")) > 0 and \
                    data.get("memoryEvents")[0].get("countInMb"):
                memory_ram = data.get("memoryEvents")[0].get("countInMb")

            if data.get("networkEvent") and data.get("networkEvent").get("wifiNetworkInfo"):
                if data.get("networkEvent").get("wifiNetworkInfo").get("linkSpeed"):
                    link_speed = data.get("networkEvent").get("wifiNetworkInfo").get("linkSpeed")

                if data.get("networkEvent").get("wifiNetworkInfo").get("signalStrength"):
                    signal_strength = data.get("networkEvent").get("wifiNetworkInfo").get("signalStrength")

        if not self.app.pargs.json:
            title = "TITLE"
            details = "DETAILS"
            renderable = [
                {title: 'battery_level', details: battery_level},
                {title: 'battery_temperature', details: battery_temp},
                {title: 'data_download', details: data_download},
                {title: 'data_upload', details: data_upload},
                {title: 'memory_storage', details: memory_storage},
                {title: 'memory_ram', details: memory_ram},
                {title: 'link_speed', details: link_speed},
                {title: 'signal_strength', details: signal_strength}
            ]

            print(f"LATEST EVENT DETAILS")
            self.app.render(renderable, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="fancy_grid")
        else:
            renderable = {
                'battery_level': battery_level,
                'battery_temperature': battery_temp,
                'data_download': data_download,
                'data_upload': data_upload,
                'memory_storage': memory_storage,
                'memory_ram': memory_ram,
                'link_speed': link_speed,
                'signal_strength': signal_strength
            }
            print(f"LATEST EVENT DETAILS")
            self.app.render(renderable, format=OutputFormat.JSON.value)
